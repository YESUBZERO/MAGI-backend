package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aires-magi/internal/handlers"
	"aires-magi/internal/models"
)

// mockSearchService implementa la interfaz requerida para búsqueda facetada y telemetría acotada
type mockSearchService struct {
	searchShipsFn      func(ctx context.Context, filter models.ShipSearchFilter) (*models.ShipSearchResponse, error)
	getShipTelemetryFn func(ctx context.Context, filter models.TelemetryQueryFilter) (*models.TelemetryQueryResponse, error)
}

func (m *mockSearchService) SearchShips(ctx context.Context, filter models.ShipSearchFilter) (*models.ShipSearchResponse, error) {
	if m.searchShipsFn != nil {
		return m.searchShipsFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockSearchService) GetShipTelemetry(ctx context.Context, filter models.TelemetryQueryFilter) (*models.TelemetryQueryResponse, error) {
	if m.getShipTelemetryFn != nil {
		return m.getShipTelemetryFn(ctx, filter)
	}
	return nil, nil
}

func setupSearchTestRouter(svc handlers.ShipSearchService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers.RegisterSearchRoutes(r, svc)
	return r
}

// -----------------------------------------------------------------------------
// CASO 2.a: GET /api/v1/ships con parámetro 'q' (búsqueda parcial case-insensitive)
// -----------------------------------------------------------------------------
func TestHandler_SearchShips_ByQueryString(t *testing.T) {
	var capturedFilter models.ShipSearchFilter

	mockSvc := &mockSearchService{
		searchShipsFn: func(ctx context.Context, filter models.ShipSearchFilter) (*models.ShipSearchResponse, error) {
			capturedFilter = filter
			return &models.ShipSearchResponse{
				Total:  2,
				Limit:  50,
				Offset: 0,
				Ships: []models.StaticAIS{
					{
						IMO:      9241061,
						MMSI:     244670000,
						Shipname: "SEASPAN EMPIRE",
						Callsign: "VRGO2",
						ShipType: "Cargo",
					},
					{
						IMO:      9241073,
						MMSI:     244670001,
						Shipname: "SEASPAN FRASER",
						Callsign: "VRGO3",
						ShipType: "Cargo",
					},
				},
			}, nil
		},
	}

	router := setupSearchTestRouter(mockSvc)

	// Búsqueda en minúsculas para verificar insensibilidad a mayúsculas
	req, err := http.NewRequest(http.MethodGet, "/api/v1/ships?q=seaspan", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "seaspan", capturedFilter.Query, "El parámetro 'q' debe capturarse en el filtro")

	var resp models.ShipSearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, int64(2), resp.Total)
	require.Len(t, resp.Ships, 2)
	assert.Contains(t, resp.Ships[0].Shipname, "SEASPAN")
	assert.Contains(t, resp.Ships[1].Shipname, "SEASPAN")
}

// -----------------------------------------------------------------------------
// CASO 2.b: GET /api/v1/ships con filtros combinados (imo, mmsi, ship_type)
// -----------------------------------------------------------------------------
func TestHandler_SearchShips_CombinedFilters(t *testing.T) {
	var capturedFilter models.ShipSearchFilter
	expectedIMO := 9241061
	expectedMMSI := 244670000
	expectedType := "Cargo"

	mockSvc := &mockSearchService{
		searchShipsFn: func(ctx context.Context, filter models.ShipSearchFilter) (*models.ShipSearchResponse, error) {
			capturedFilter = filter
			return &models.ShipSearchResponse{
				Total:  1,
				Limit:  50,
				Offset: 0,
				Ships: []models.StaticAIS{
					{
						IMO:      expectedIMO,
						MMSI:     expectedMMSI,
						Shipname: "SEASPAN EMPIRE",
						ShipType: expectedType,
					},
				},
			}, nil
		},
	}

	router := setupSearchTestRouter(mockSvc)

	url := fmt.Sprintf("/api/v1/ships?imo=%d&mmsi=%d&ship_type=%s", expectedIMO, expectedMMSI, expectedType)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedFilter.IMO)
	assert.Equal(t, expectedIMO, *capturedFilter.IMO)
	require.NotNil(t, capturedFilter.MMSI)
	assert.Equal(t, expectedMMSI, *capturedFilter.MMSI)
	assert.Equal(t, expectedType, capturedFilter.ShipType)

	var resp models.ShipSearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, int64(1), resp.Total)
	assert.Equal(t, expectedMMSI, resp.Ships[0].MMSI)
}

// -----------------------------------------------------------------------------
// CASO 2.c: GET /api/v1/ships con parámetros de paginación (limit, offset)
// -----------------------------------------------------------------------------
func TestHandler_SearchShips_Pagination(t *testing.T) {
	var capturedFilter models.ShipSearchFilter

	mockSvc := &mockSearchService{
		searchShipsFn: func(ctx context.Context, filter models.ShipSearchFilter) (*models.ShipSearchResponse, error) {
			capturedFilter = filter
			return &models.ShipSearchResponse{
				Total:  125,
				Limit:  filter.Limit,
				Offset: filter.Offset,
				Ships:  make([]models.StaticAIS, filter.Limit),
			}, nil
		},
	}

	router := setupSearchTestRouter(mockSvc)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/ships?limit=15&offset=30", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 15, capturedFilter.Limit)
	assert.Equal(t, 30, capturedFilter.Offset)

	var resp models.ShipSearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, int64(125), resp.Total)
	assert.Equal(t, 15, resp.Limit)
	assert.Equal(t, 30, resp.Offset)
	assert.Len(t, resp.Ships, 15)
}

// -----------------------------------------------------------------------------
// CASO 2.d: GET /api/v1/ships/:mmsi/telemetry con rango de fechas ISO8601 ('from' y 'to')
// -----------------------------------------------------------------------------
func TestHandler_GetShipTelemetry_DateRange(t *testing.T) {
	var capturedFilter models.TelemetryQueryFilter

	fromTime, _ := time.Parse(time.RFC3339, "2026-10-20T00:00:00Z")
	toTime, _ := time.Parse(time.RFC3339, "2026-10-23T23:59:59Z")
	targetMMSI := 244670000

	mockSvc := &mockSearchService{
		getShipTelemetryFn: func(ctx context.Context, filter models.TelemetryQueryFilter) (*models.TelemetryQueryResponse, error) {
			capturedFilter = filter
			return &models.TelemetryQueryResponse{
				MMSI:  filter.MMSI,
				From:  filter.From,
				To:    filter.To,
				Total: 2,
				Telemetry: []models.DynamicHistoryResponse{
					{
						Timestamp: fromTime.Add(2 * time.Hour),
						Latitude:  24.555,
						Longitude: -81.782,
						Status:    "Underway Using Engine",
						Speed:     15.4,
						Course:    182.0,
					},
					{
						Timestamp: fromTime.Add(4 * time.Hour),
						Latitude:  24.810,
						Longitude: -81.950,
						Status:    "Underway Using Engine",
						Speed:     15.8,
						Course:    184.5,
					},
				},
			}, nil
		},
	}

	router := setupSearchTestRouter(mockSvc)

	url := fmt.Sprintf("/api/v1/ships/%d/telemetry?from=%s&to=%s", targetMMSI, fromTime.Format(time.RFC3339), toTime.Format(time.RFC3339))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, targetMMSI, capturedFilter.MMSI)
	assert.True(t, fromTime.Equal(capturedFilter.From), "La fecha 'from' debe coincidir")
	assert.True(t, toTime.Equal(capturedFilter.To), "La fecha 'to' debe coincidir")

	var resp models.TelemetryQueryResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, targetMMSI, resp.MMSI)
	assert.Equal(t, int64(2), resp.Total)
	require.Len(t, resp.Telemetry, 2)
	assert.Equal(t, 15.4, resp.Telemetry[0].Speed)
	assert.Equal(t, 15.8, resp.Telemetry[1].Speed)
}

// -----------------------------------------------------------------------------
// CASOS DE BORDE (EDGE CASES)
// -----------------------------------------------------------------------------
func TestHandler_GetShipTelemetry_InvalidInputs(t *testing.T) {
	mockSvc := &mockSearchService{}
	router := setupSearchTestRouter(mockSvc)

	t.Run("MMSI invalido no numerico -> 400 Bad Request", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/ships/not-a-number/telemetry", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Fecha 'from' no cumple formato ISO8601 -> 400 Bad Request", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/ships/244670000/telemetry?from=2026-99-99&to=2026-10-23T00:00:00Z", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Fecha 'from' es posterior a 'to' -> 400 Bad Request", func(t *testing.T) {
		from := "2026-10-25T00:00:00Z"
		to := "2026-10-20T00:00:00Z"
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/ships/244670000/telemetry?from=%s&to=%s", from, to), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
