package ais_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YESUBZERO/consumer-service/internal/ais"
	"github.com/gin-gonic/gin"
)

// mockService implementa ais.Service para probar ais.Handler
type mockService struct {
	processStaticFn  func(msg *ais.StaticAIS) error
	processDynamicFn func(msg *ais.DynamicAIS) error
	getShipByIMOFn   func(imo int, limit, offset int) (*ais.StaticAIS, error)
	getShipByMMSIFn  func(mmsi int, limit, offset int) (*ais.StaticAIS, error)
}

func (m *mockService) ProcessStaticMessage(msg *ais.StaticAIS) error {
	if m.processStaticFn != nil {
		return m.processStaticFn(msg)
	}
	return nil
}

func (m *mockService) ProcessDynamicMessage(msg *ais.DynamicAIS) error {
	if m.processDynamicFn != nil {
		return m.processDynamicFn(msg)
	}
	return nil
}

func (m *mockService) GetShipByIMO(imo int, limit, offset int) (*ais.StaticAIS, error) {
	if m.getShipByIMOFn != nil {
		return m.getShipByIMOFn(imo, limit, offset)
	}
	return nil, nil
}

func (m *mockService) GetShipByMMSI(mmsi int, limit, offset int) (*ais.StaticAIS, error) {
	if m.getShipByMMSIFn != nil {
		return m.getShipByMMSIFn(mmsi, limit, offset)
	}
	return nil, nil
}

func setupRouter(svc ais.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ais.NewHandler(r, svc, svc)
	return r
}

func TestHandler_CreateStaticMessage_Success(t *testing.T) {
	svc := &mockService{
		processStaticFn: func(msg *ais.StaticAIS) error {
			return nil
		},
	}
	r := setupRouter(svc)

	payload := map[string]interface{}{
		"msg_type":  5,
		"imo":       9241061,
		"mmsi":      244670000,
		"shipname":  "SEASPAN EMPIRE",
		"callsign":  "VRGO2",
		"ship_type": "Cargo",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/static", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("se esperaba status 201, se obtuvo: %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_CreateStaticMessage_BadRequest(t *testing.T) {
	r := setupRouter(&mockService{})

	// Faltan campos requeridos según tag binding:"required"
	payload := map[string]interface{}{
		"msg_type": 5,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/static", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba status 400, se obtuvo: %d", w.Code)
	}
}

func TestHandler_CreateStaticMessage_ServiceError(t *testing.T) {
	svc := &mockService{
		processStaticFn: func(msg *ais.StaticAIS) error {
			return errors.New("db error")
		},
	}
	r := setupRouter(svc)

	payload := map[string]interface{}{
		"msg_type":  5,
		"imo":       9241061,
		"mmsi":      244670000,
		"shipname":  "SEASPAN EMPIRE",
		"callsign":  "VRGO2",
		"ship_type": "Cargo",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/static", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("se esperaba status 500, se obtuvo: %d", w.Code)
	}
}

func TestHandler_GetShipByIMO_Success(t *testing.T) {
	svc := &mockService{
		getShipByIMOFn: func(imo int, limit, offset int) (*ais.StaticAIS, error) {
			return &ais.StaticAIS{
				IMO:      9241061,
				MMSI:     244670000,
				Shipname: "SEASPAN EMPIRE",
				Dynamics: []ais.DynamicAIS{
					{
						MMSI:      244670000,
						Timestamp: time.Now(),
						Latitude:  12.34,
						Longitude: -76.54,
						Speed:     12.0,
						Course:    180.0,
					},
				},
			}, nil
		},
	}
	r := setupRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/ship/9241061?limit=25&offset=5", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200, se obtuvo: %d", w.Code)
	}

	var res ais.ShipDetailsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("error decodificando respuesta: %v", err)
	}
	if res.IMO != 9241061 || res.TotalSpots != 1 {
		t.Errorf("respuesta inconsistente: %+v", res)
	}
}

func TestHandler_GetShipByIMO_NotFound(t *testing.T) {
	svc := &mockService{
		getShipByIMOFn: func(imo int, limit, offset int) (*ais.StaticAIS, error) {
			return nil, ais.ErrShipNotFound
		},
	}
	r := setupRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/ship/9241061", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("se esperaba status 404 para ErrShipNotFound, se obtuvo: %d", w.Code)
	}
}

func TestHandler_GetShipByIMO_InternalError(t *testing.T) {
	svc := &mockService{
		getShipByIMOFn: func(imo int, limit, offset int) (*ais.StaticAIS, error) {
			return nil, errors.New("database connection refused")
		},
	}
	r := setupRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/ship/9241061", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("se esperaba status 500 para error de DB, se obtuvo: %d", w.Code)
	}
}

func TestHandler_GetShipByMMSI_Success(t *testing.T) {
	svc := &mockService{
		getShipByMMSIFn: func(mmsi int, limit, offset int) (*ais.StaticAIS, error) {
			return &ais.StaticAIS{
				IMO:      9241061,
				MMSI:     244670000,
				Shipname: "SEASPAN EMPIRE",
				Dynamics: []ais.DynamicAIS{},
			}, nil
		},
	}
	r := setupRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/ship/mmsi/244670000", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200, se obtuvo: %d", w.Code)
	}
}

func TestHandler_GetShipByIMO_InvalidParam(t *testing.T) {
	r := setupRouter(&mockService{})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/ship/not-a-number", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("se esperaba status 400 para parámetro IMO no numérico, se obtuvo: %d", w.Code)
	}
}

func TestHandler_QueryOnlyMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	querySvc := &mockService{
		getShipByIMOFn: func(imo int, limit, offset int) (*ais.StaticAIS, error) {
			return &ais.StaticAIS{IMO: 9241061, Shipname: "SEASPAN EMPIRE"}, nil
		},
	}
	// Solo inyectamos QueryService (sin IngestionService)
	ais.NewHandler(r, querySvc)

	// Verificamos que la ruta de consulta funcione
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/ship/9241061", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("se esperaba status 200 en modo solo consulta, se obtuvo: %d", w.Code)
	}

	// Verificamos que la ruta de ingesta POST /static no esté registrada (retorna 404)
	reqPost, _ := http.NewRequest(http.MethodPost, "/api/v1/static", bytes.NewBufferString("{}"))
	wPost := httptest.NewRecorder()
	r.ServeHTTP(wPost, reqPost)
	if wPost.Code != http.StatusNotFound {
		t.Fatalf("se esperaba status 404 para POST /static en modo solo consulta, se obtuvo: %d", wPost.Code)
	}
}

func TestHandler_IngestionOnlyMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ingestionSvc := &mockService{
		processStaticFn: func(msg *ais.StaticAIS) error {
			return nil
		},
	}
	// Inyectamos nil para QueryService y proveemos IngestionService
	ais.NewHandler(r, nil, ingestionSvc)

	// Verificamos que la ruta de ingesta POST /static funcione
	payload := map[string]interface{}{
		"msg_type":  5,
		"imo":       9241061,
		"mmsi":      244670000,
		"shipname":  "SEASPAN EMPIRE",
		"callsign":  "VRGO2",
		"ship_type": "Cargo",
	}
	body, _ := json.Marshal(payload)
	reqPost, _ := http.NewRequest(http.MethodPost, "/api/v1/static", bytes.NewBuffer(body))
	reqPost.Header.Set("Content-Type", "application/json")
	wPost := httptest.NewRecorder()
	r.ServeHTTP(wPost, reqPost)
	if wPost.Code != http.StatusCreated {
		t.Fatalf("se esperaba status 201 en modo ingesta, se obtuvo: %d", wPost.Code)
	}

	// Verificamos que la ruta de consulta GET /ship/9241061 no esté registrada (retorna 404)
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/v1/ship/9241061", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusNotFound {
		t.Fatalf("se esperaba status 404 para GET /ship/:imo en modo solo ingesta, se obtuvo: %d", wGet.Code)
	}
}
