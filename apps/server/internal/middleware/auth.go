package middleware

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

const (
	// ContextKeyUserID clave para el identificador del usuario extraído del claim sub
	ContextKeyUserID = "userID"
	// ContextKeyRoles clave para la lista de roles RBAC del usuario
	ContextKeyRoles = "roles"
	// ContextKeyClaims clave para el objeto completo de claims de Zitadel
	ContextKeyClaims = "claims"
	// ZitadelRoleClaimKey URN estándar de Zitadel para roles de proyecto
	ZitadelRoleClaimKey = "urn:zitadel:iam:org:project:roles"
)

// ZitadelConfig almacena la configuración de validación de tokens de Zitadel
type ZitadelConfig struct {
	Issuer   string
	JWKSURL  string
	Audience string
}

// ZitadelClaims representa las aserciones de identidad y roles emitidas por Zitadel OIDC
type ZitadelClaims struct {
	Issuer    string                       `json:"iss"`
	Subject   string                       `json:"sub"`
	Audience  interface{}                  `json:"aud"`
	ExpiresAt int64                        `json:"exp"`
	IssuedAt  int64                        `json:"iat"`
	RolesMap  map[string]map[string]string `json:"urn:zitadel:iam:org:project:roles,omitempty"`
	RolesList []string                     `json:"roles,omitempty"`
}

// AuthMiddleware define el contrato para autenticación y autorización RBAC
type AuthMiddleware interface {
	RequireAuth() gin.HandlerFunc
	RequireRole(role string) gin.HandlerFunc
}

type jwksResponse struct {
	Keys []struct {
		Kty string `json:"kty"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

// zitadelMiddleware implementación base del middleware de autenticación
type zitadelMiddleware struct {
	cfg         ZitadelConfig
	httpClient  *http.Client
	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	sf          singleflight.Group
	lastRefresh time.Time
	cooldown    time.Duration
}

// NewZitadelAuthMiddleware construye el middleware de autenticación Zitadel
func NewZitadelAuthMiddleware(cfg ZitadelConfig) AuthMiddleware {
	return &zitadelMiddleware{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		keys:       make(map[string]*rsa.PublicKey),
		cooldown:   1 * time.Minute,
	}
}

// RequireAuth valida el Bearer token JWT e inyecta la identidad en el contexto
func (m *zitadelMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "cabecera Authorization ausente",
			})
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "formato de token inválido, se requiere prefijo Bearer",
			})
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token Bearer vacío",
			})
			return
		}

		parts := strings.Split(tokenStr, ".")
		if len(parts) != 3 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "estructura de token JWT inválida",
			})
			return
		}

		// Decodificar cabecera JWT
		headerBytes, err := decodeBase64URL(parts[0])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "cabecera de token inválida",
			})
			return
		}

		var hdr jwtHeader
		if err := json.Unmarshal(headerBytes, &hdr); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "cabecera de token malformada",
			})
			return
		}

		if hdr.Alg != "RS256" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "algoritmo de firma no soportado",
			})
			return
		}

		// Obtener clave pública RSA por kid
		pubKey, err := m.getKey(hdr.Kid)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "clave pública no encontrada",
			})
			return
		}

		// Validar firma criptográfica RS256
		sigBytes, err := decodeBase64URL(parts[2])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "firma de token malformada",
			})
			return
		}

		signedData := parts[0] + "." + parts[1]
		hashed := sha256.Sum256([]byte(signedData))
		if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], sigBytes); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "firma de token inválida",
			})
			return
		}

		// Decodificar claims
		claimsBytes, err := decodeBase64URL(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "claims de token inválidos",
			})
			return
		}

		var claims ZitadelClaims
		if err := json.Unmarshal(claimsBytes, &claims); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "claims de token malformados",
			})
			return
		}

		// Validar expiración (exp) obligatoria y vigencia
		if claims.ExpiresAt == 0 || time.Now().Unix() > claims.ExpiresAt {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token expirado o sin tiempo de expiración",
			})
			return
		}

		// Validar emisor (iss)
		if m.cfg.Issuer != "" && claims.Issuer != m.cfg.Issuer {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "emisor del token no coincide",
			})
			return
		}

		// Validar audiencia (aud) obligatoria si está configurada
		if m.cfg.Audience != "" {
			if claims.Audience == nil || !validateAudience(claims.Audience, m.cfg.Audience) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "audiencia del token no coincide o está ausente",
				})
				return
			}
		}

		// Extraer roles normalizados
		rolesMap := make(map[string]bool)
		for r := range claims.RolesMap {
			rolesMap[r] = true
		}
		for _, r := range claims.RolesList {
			rolesMap[r] = true
		}
		roles := make([]string, 0, len(rolesMap))
		for r := range rolesMap {
			roles = append(roles, r)
		}

		// Inyectar contexto
		c.Set(ContextKeyUserID, claims.Subject)
		c.Set(ContextKeyRoles, roles)
		c.Set(ContextKeyClaims, claims)

		c.Next()
	}
}

// RequireRole verifica que el usuario autenticado cuente con el rol RBAC requerido
func (m *zitadelMiddleware) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasRole(c, role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "acceso denegado: permisos insuficientes",
			})
			return
		}
		c.Next()
	}
}

// getKey recupera la clave pública RSA de la caché o del endpoint JWKS
func (m *zitadelMiddleware) getKey(kid string) (*rsa.PublicKey, error) {
	if kid == "" {
		return nil, errors.New("kid vacío")
	}

	m.mu.RLock()
	key, exists := m.keys[kid]
	m.mu.RUnlock()
	if exists {
		return key, nil
	}

	// Colapsar solicitudes concurrentes al JWKS mediante singleflight
	_, err, _ := m.sf.Do("refresh_jwks", func() (interface{}, error) {
		return nil, m.refreshKeys()
	})
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	key, exists = m.keys[kid]
	m.mu.RUnlock()
	if exists {
		return key, nil
	}

	return nil, fmt.Errorf("clave pública con kid '%s' no encontrada", kid)
}

// refreshKeys consulta el endpoint JWKS y almacena en caché las claves públicas encontradas
func (m *zitadelMiddleware) refreshKeys() error {
	m.mu.Lock()
	if !m.lastRefresh.IsZero() && time.Since(m.lastRefresh) < m.cooldown {
		m.mu.Unlock()
		return nil // Cooldown activo: evita saturar a Zitadel con 'kid' desconocidos
	}
	m.mu.Unlock()

	if m.cfg.JWKSURL == "" {
		return errors.New("JWKS URL no configurada")
	}

	resp, err := m.httpClient.Get(m.cfg.JWKSURL)
	if err != nil {
		return fmt.Errorf("error al consultar endpoint JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("el endpoint JWKS respondió con código HTTP %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("error al parsear JSON del JWKS: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pubKey, err := parseRSAPublicKey(k.N, k.E)
		if err == nil && k.Kid != "" {
			m.keys[k.Kid] = pubKey
		}
	}
	m.lastRefresh = time.Now()

	return nil
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := decodeBase64URL(nStr)
	if err != nil {
		return nil, fmt.Errorf("error decodificando modulus (n): %w", err)
	}

	eBytes, err := decodeBase64URL(eStr)
	if err != nil {
		return nil, fmt.Errorf("error decodificando exponent (e): %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}

func decodeBase64URL(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return base64.URLEncoding.DecodeString(s)
	}
	return b, nil
}

func validateAudience(aud interface{}, expected string) bool {
	switch v := aud.(type) {
	case string:
		return v == expected
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == expected {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s == expected {
				return true
			}
		}
	}
	return false
}

// GetUserID extrae el subject del usuario desde el contexto de Gin
func GetUserID(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextKeyUserID)
	if !exists {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

// GetRoles extrae la lista de roles asignados desde el contexto de Gin
func GetRoles(c *gin.Context) ([]string, bool) {
	val, exists := c.Get(ContextKeyRoles)
	if !exists {
		return nil, false
	}
	roles, ok := val.([]string)
	return roles, ok
}

// HasRole verifica si un rol específico está presente en el contexto de Gin
func HasRole(c *gin.Context, role string) bool {
	roles, ok := GetRoles(c)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
