package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aires-magi/internal/ais"
	"aires-magi/internal/handlers"
	"aires-magi/internal/models"
)

type mockAISService struct {
	processStaticFn  func(msg *models.StaticAIS) error
	processDynamicFn func(msg *models.DynamicAIS) error
	getShipByIMOFn   func(imo int, limit, offset int) (*models.StaticAIS, error)
	getShipByMMSIFn  func(mmsi int, limit, offset int) (*models.StaticAIS, error)
}

func (m *mockAISService) ProcessStaticMessage(msg *models.StaticAIS) error {
	if m.processStaticFn != nil {
		return m.processStaticFn(msg)
	}
	return nil
}

func (m *mockAISService) ProcessDynamicMessage(msg *models.DynamicAIS) error {
	if m.processDynamicFn != nil {
		return m.processDynamicFn(msg)
	}
	return nil
}

func (m *mockAISService) GetShipByIMO(imo int, limit, offset int) (*models.StaticAIS, error) {
	if m.getShipByIMOFn != nil {
		return m.getShipByIMOFn(imo, limit, offset)
	}
	return nil, nil
}

func (m *mockAISService) GetShipByMMSI(mmsi int, limit, offset int) (*models.StaticAIS, error) {
	if m.getShipByMMSIFn != nil {
		return m.getShipByMMSIFn(mmsi, limit, offset)
	}
	return nil, nil
}

func setupTestRouter(svc ais.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	logger := zerolog.Nop()
	handlers.NewAISHandler(r, svc, svc, nil, logger)
	return r
}

func TestHandler_CreateStaticMessage_Success(t *testing.T) {
	svc := &mockAISService{
		processStaticFn: func(msg *models.StaticAIS) error {
			return nil
		},
	}
	r := setupTestRouter(svc)

	payload := models.CreateStaticMessageRequest{
		MsgType:  5,
		IMO:      9241061,
		MMSI:     244670000,
		Shipname: "SEASPAN EMPIRE",
		Callsign: "VRGO2",
		ShipType: "Cargo",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/static", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_GetShipByIMO_Success(t *testing.T) {
	svc := &mockAISService{
		getShipByIMOFn: func(imo int, limit, offset int) (*models.StaticAIS, error) {
			return &models.StaticAIS{
				IMO:      imo,
				MMSI:     244670000,
				Shipname: "SEASPAN EMPIRE",
				Dynamics: []models.DynamicAIS{
					{
						Timestamp: time.Now(),
						Latitude:  -33.0245,
						Longitude: -71.6212,
						Speed:     12.5,
						Course:    180.0,
					},
				},
			}, nil
		},
	}
	r := setupTestRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/ship/9241061", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var res models.ShipDetailsResponse
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.Equal(t, 9241061, res.IMO)
	assert.Equal(t, 1, res.TotalSpots)
}

func TestHandler_HealthCheck(t *testing.T) {
	r := setupTestRouter(&mockAISService{})

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
