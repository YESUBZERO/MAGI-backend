package middleware_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aires-magi/internal/middleware"
)

// Helpers para generación de llaves RSA, JWKS y tokens JWT de prueba sin servidor externo
type testJWTSuite struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keyID      string
	jwksServer *httptest.Server
	issuer     string
}

func setupTestJWTSuite(t *testing.T) *testJWTSuite {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyID := "test-key-id-magi-2026"
	pubKey := &privKey.PublicKey

	// Servir JWKS simulando el endpoint OIDC de Zitadel: /.well-known/jwks.json
	nStr := base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes())
	eBytes := big.NewInt(int64(pubKey.E)).Bytes()
	eStr := base64.RawURLEncoding.EncodeToString(eBytes)

	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": keyID,
				"n":   nStr,
				"e":   eStr,
			},
		},
	}
	jwksJSON, err := json.Marshal(jwks)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jwksJSON)
	}))

	return &testJWTSuite{
		privateKey: privKey,
		publicKey:  pubKey,
		keyID:      keyID,
		jwksServer: server,
		issuer:     server.URL,
	}
}

func (s *testJWTSuite) tearDown() {
	if s.jwksServer != nil {
		s.jwksServer.Close()
	}
}

// createSignedToken genera un JWT válido o expirado firmado con RS256
func (s *testJWTSuite) createSignedToken(
	t *testing.T,
	key *rsa.PrivateKey,
	keyID string,
	subject string,
	roles []string,
	expiration time.Duration,
) string {
	return s.createSignedTokenWithAud(t, key, keyID, subject, roles, expiration, "magi-backend")
}

func (s *testJWTSuite) createSignedTokenWithAud(
	t *testing.T,
	key *rsa.PrivateKey,
	keyID string,
	subject string,
	roles []string,
	expiration time.Duration,
	audience interface{},
) string {
	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
		"kid": keyID,
	}

	now := time.Now()
	rolesMap := make(map[string]map[string]string)
	for _, r := range roles {
		rolesMap[r] = map[string]string{"302948293": "magi-org"}
	}

	claims := map[string]interface{}{
		"iss":                               s.issuer,
		"sub":                               subject,
		"exp":                               now.Add(expiration).Unix(),
		"iat":                               now.Unix(),
		"urn:zitadel:iam:org:project:roles": rolesMap,
		"roles":                             roles,
	}
	if audience != nil {
		claims["aud"] = audience
	}

	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)
	claimsJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := encodedHeader + "." + encodedClaims

	hashed := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	require.NoError(t, err)

	encodedSig := base64.RawURLEncoding.EncodeToString(sig)
	return signingInput + "." + encodedSig
}

func setupTestRouterWithAuth(auth middleware.AuthMiddleware, requiredRole string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	api := r.Group("/api/v1")
	api.Use(auth.RequireAuth())
	if requiredRole != "" {
		api.Use(auth.RequireRole(requiredRole))
	}

	api.GET("/protected", func(c *gin.Context) {
		userID, _ := middleware.GetUserID(c)
		roles, _ := middleware.GetRoles(c)
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"user_id": userID,
			"roles":   roles,
		})
	})

	return r
}

func TestZitadelAuthMiddleware_TableDriven(t *testing.T) {
	jwtSuite := setupTestJWTSuite(t)
	defer jwtSuite.tearDown()

	// Llave ajena no registrada para simular firma inválida
	untrustedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	authCfg := middleware.ZitadelConfig{
		Issuer:   jwtSuite.issuer,
		JWKSURL:  jwtSuite.jwksServer.URL,
		Audience: "magi-backend",
	}
	authMW := middleware.NewZitadelAuthMiddleware(authCfg)

	// Generar tokens para los distintos escenarios de prueba
	validViewerToken := jwtSuite.createSignedToken(t, jwtSuite.privateKey, jwtSuite.keyID, "usr-viewer-123", []string{"ais:viewer"}, time.Hour)
	validExportToken := jwtSuite.createSignedToken(t, jwtSuite.privateKey, jwtSuite.keyID, "usr-export-456", []string{"ais:export"}, time.Hour)
	expiredToken := jwtSuite.createSignedToken(t, jwtSuite.privateKey, jwtSuite.keyID, "usr-expired-999", []string{"ais:viewer"}, -1*time.Hour)
	invalidSigToken := jwtSuite.createSignedToken(t, untrustedKey, "unknown-kid", "usr-intruder", []string{"ais:viewer"}, time.Hour)
	insufficientRoleToken := jwtSuite.createSignedToken(t, jwtSuite.privateKey, jwtSuite.keyID, "usr-basic", []string{"other:role"}, time.Hour)
	missingAudToken := jwtSuite.createSignedTokenWithAud(t, jwtSuite.privateKey, jwtSuite.keyID, "usr-no-aud", []string{"ais:viewer"}, time.Hour, nil)
	wrongAudToken := jwtSuite.createSignedTokenWithAud(t, jwtSuite.privateKey, jwtSuite.keyID, "usr-wrong-aud", []string{"ais:viewer"}, time.Hour, "other-backend")

	tests := []struct {
		name           string
		requiredRole   string
		authHeader     string
		expectedStatus int
		expectedSub    string
		validateCtx    bool
	}{
		{
			name:           "a) Peticion sin header Authorization -> 401 Unauthorized",
			requiredRole:   "ais:viewer",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "b1) Peticion con header malformado (sin prefijo Bearer) -> 401 Unauthorized",
			requiredRole:   "ais:viewer",
			authHeader:     "Basic dXNlcjpwYXNz",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "b2) Peticion con Bearer incompleto -> 401 Unauthorized",
			requiredRole:   "ais:viewer",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "b3) Peticion con token malformado sintacticamente -> 401 Unauthorized",
			requiredRole:   "ais:viewer",
			authHeader:     "Bearer not.a.valid.jwt",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "c1) Peticion con token expirado -> 401 Unauthorized",
			requiredRole:   "ais:viewer",
			authHeader:     "Bearer " + expiredToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "c2) Peticion con firma RSA invalida / kid desconocido -> 401 Unauthorized",
			requiredRole:   "ais:viewer",
			authHeader:     "Bearer " + invalidSigToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "c3) Peticion con token sin claim 'aud' cuando se requiere audiencia -> 401 Unauthorized",
			requiredRole:   "ais:viewer",
			authHeader:     "Bearer " + missingAudToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "c4) Peticion con token con 'aud' no coincidente -> 401 Unauthorized",
			requiredRole:   "ais:viewer",
			authHeader:     "Bearer " + wrongAudToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "d1) Peticion con token valido pero sin rol 'ais:viewer' requerido -> 403 Forbidden",
			requiredRole:   "ais:viewer",
			authHeader:     "Bearer " + insufficientRoleToken,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "d2) Peticion con token 'ais:viewer' intentando acceder a ruta con rol 'ais:export' -> 403 Forbidden",
			requiredRole:   "ais:export",
			authHeader:     "Bearer " + validViewerToken,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "e1) Peticion con token valido y rol 'ais:viewer' correcto -> 200 OK y contexto de usuario inyectado",
			requiredRole:   "ais:viewer",
			authHeader:     "Bearer " + validViewerToken,
			expectedStatus: http.StatusOK,
			expectedSub:    "usr-viewer-123",
			validateCtx:    true,
		},
		{
			name:           "e2) Peticion con token valido y rol 'ais:export' correcto -> 200 OK y contexto de usuario inyectado",
			requiredRole:   "ais:export",
			authHeader:     "Bearer " + validExportToken,
			expectedStatus: http.StatusOK,
			expectedSub:    "usr-export-456",
			validateCtx:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouterWithAuth(authMW, tt.requiredRole)

			req, err := http.NewRequest(http.MethodGet, "/api/v1/protected", nil)
			require.NoError(t, err)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "El código de estado HTTP debe coincidir con el contrato RBAC")

			if tt.validateCtx && w.Code == http.StatusOK {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedSub, resp["user_id"], "El sub / userID extraído del JWT debe inyectarse en el contexto")
			}
		})
	}
}
