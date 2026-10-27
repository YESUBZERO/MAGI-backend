package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"aires-magi/internal/models"
)

// ShipSearchService define las operaciones de negocio para búsqueda facetada y telemetría acotada
type ShipSearchService interface {
	SearchShips(ctx context.Context, filter models.ShipSearchFilter) (*models.ShipSearchResponse, error)
	GetShipTelemetry(ctx context.Context, filter models.TelemetryQueryFilter) (*models.TelemetryQueryResponse, error)
}

// SearchHandler expone los endpoints de búsqueda facetada de buques y telemetría
type SearchHandler struct {
	service ShipSearchService
}

// NewSearchHandler crea una instancia del SearchHandler
func NewSearchHandler(svc ShipSearchService) *SearchHandler {
	return &SearchHandler{service: svc}
}

// RegisterSearchRoutes registra los endpoints de búsqueda en el motor de Gin
func RegisterSearchRoutes(r *gin.Engine, svc ShipSearchService) *SearchHandler {
	h := NewSearchHandler(svc)
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ships", h.SearchShips)
		v1.GET("/ships/:mmsi/telemetry", h.GetShipTelemetry)
	}
	return h
}

// SearchShips gestiona la búsqueda facetada por nombre parcial insensible a mayúsculas, filtros combinados y paginación
// GET /api/v1/ships
func (h *SearchHandler) SearchShips(c *gin.Context) {
	var filter models.ShipSearchFilter

	filter.Query = c.Query("q")
	filter.ShipType = c.Query("ship_type")

	if imoStr := c.Query("imo"); imoStr != "" {
		if imoVal, err := strconv.Atoi(imoStr); err == nil {
			filter.IMO = &imoVal
		}
	}

	if mmsiStr := c.Query("mmsi"); mmsiStr != "" {
		if mmsiVal, err := strconv.Atoi(mmsiStr); err == nil {
			filter.MMSI = &mmsiVal
		}
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}
	filter.Limit = limit

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}
	filter.Offset = offset

	if h.service == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "servicio de búsqueda no inicializado"})
		return
	}

	resp, err := h.service.SearchShips(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetShipTelemetry gestiona la consulta de historial de telemetría acotada por rango temporal ISO8601
// GET /api/v1/ships/:mmsi/telemetry
func (h *SearchHandler) GetShipTelemetry(c *gin.Context) {
	mmsiParam := c.Param("mmsi")
	mmsi, err := strconv.Atoi(mmsiParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el MMSI debe ser un número entero válido"})
		return
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")

	fromTime, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el parámetro 'from' debe estar en formato ISO8601 (RFC3339)"})
		return
	}

	toTime, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el parámetro 'to' debe estar en formato ISO8601 (RFC3339)"})
		return
	}

	if fromTime.After(toTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la fecha 'from' no puede ser posterior a 'to'"})
		return
	}

	filter := models.TelemetryQueryFilter{
		MMSI: mmsi,
		From: fromTime,
		To:   toTime,
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filter.Limit = limit
	filter.Offset = offset

	if h.service == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "servicio de telemetría no inicializado"})
		return
	}

	resp, err := h.service.GetShipTelemetry(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
