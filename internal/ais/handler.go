package ais

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ===========================================================
// MODELOS DE REQUEST Y RESPONSE (DTOs)
// ===========================================================

// CreateStaticRequest es lo que espera recibit el API al registrar un barco
type CreateStaticMessageRequest struct {
	MsgType  int    `json:"msg_type" binding:"required"`
	IMO      int    `json:"imo" binding:"required"`
	MMSI     int    `json:"mmsi" binding:"required"`
	Callsign string `json:"callsign"`
	Shipname string `json:"shipname" binding:"required"`
	ShipType string `json:"ship_type"`
}

// ShipDetailsResponse retorna al cliente la info completa de un barco.
type ShipDetailsResponse struct {
	IMO        int                      `json:"imo"`
	MMSI       int                      `json:"mmsi"`
	Callsign   string                   `json:"callsign"`
	Shipname   string                   `json:"shipname"`
	ShipType   string                   `json:"ship_type"`
	TotalSpots int                      `json:"total_positions_recorded"`
	History    []DynamicHistoryResponse `json:"position_history,omitempty"`
}

// DynamicHistoryResponse returna el historial de movimientos de un barco
type DynamicHistoryResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Status    string    `json:"status"`
	Speed     float64   `json:"speed_knots"`
	Course    float64   `json:"course"`
}

// =========================================================================
// CONTROLADOR / HANDLER
// =========================================================================

// Definimos los atributos del handler
type Handler struct {
	queryService     QueryService
	ingestionService IngestionService
}

func NewHandler(r *gin.Engine, q QueryService, ing ...IngestionService) *Handler {
	h := &Handler{queryService: q}
	if len(ing) > 0 {
		h.ingestionService = ing[0]
	}

	routes := r.Group("/api/v1")
	{
		if h.ingestionService != nil {
			routes.POST("/static", h.CreateStaticMessage)
		}
		if h.queryService != nil {
			routes.GET("/ship/:imo", h.GetShipByIMO)
			routes.GET("/ship/mmsi/:mmsi", h.GetShipByMMSI)
		}
	}
	return h
}

// CreateStaticMessage maneja la creación/actualizacion via HTTP
func (h *Handler) CreateStaticMessage(c *gin.Context) {
	var req CreateStaticMessageRequest

	// 1. Validar las propiedades del request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return
	}

	// 2. Mapear el DTO de Reques a la Entidad de Dominio Nativa
	msg := &StaticAIS{
		MsgType:  req.MsgType,
		IMO:      req.IMO,
		MMSI:     req.MMSI,
		Callsign: req.Callsign,
		Shipname: req.Shipname,
		ShipType: req.ShipType,
	}

	// 3. Delegamos al servicio ProcessStaticMessage
	if h.ingestionService == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "módulo de ingesta no disponible"})
		return
	}
	if err := h.ingestionService.ProcessStaticMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Datos estáticos del buque procesados correctamente"})

}

func (h *Handler) GetShipByIMO(c *gin.Context) {

	// 1. Validamos el parametro recibido de la ruta
	imoParam := c.Param("imo")
	imo, err := strconv.Atoi(imoParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el IMO debe ser un número válido"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 2. Llamar al servicio GetShipByIMO
	if h.queryService == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "módulo de consultas no disponible"})
		return
	}
	ship, err := h.queryService.GetShipByIMO(imo, limit, offset)
	if err != nil {
		if errors.Is(err, ErrShipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno consultando el buque: " + err.Error()})
		return
	}
	// 3. Mapear la entidad de dominio al DTO de Response
	historyResponse := make([]DynamicHistoryResponse, len(ship.Dynamics))
	for i, d := range ship.Dynamics {
		historyResponse[i] = DynamicHistoryResponse{
			Timestamp: d.Timestamp,
			Longitude: d.Longitude,
			Latitude:  d.Latitude,
			Status:    d.Status,
			Speed:     d.Speed,
			Course:    d.Course,
		}
	}

	res := ShipDetailsResponse{
		IMO:        ship.IMO,
		MMSI:       ship.MMSI,
		Callsign:   ship.Callsign,
		Shipname:   ship.Shipname,
		ShipType:   ship.ShipType,
		TotalSpots: len(ship.Dynamics),
		History:    historyResponse,
	}

	// 4. Confirmamos la respuesta status 200
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetShipByMMSI(c *gin.Context) {

	// 1. Validamos el parametro recibido de la ruta
	mmsiParam := c.Param("mmsi")
	mmsi, err := strconv.Atoi(mmsiParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el MMSI debe ser un número válido"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 2. Llamar al servicio GetShipByMMSI
	if h.queryService == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "módulo de consultas no disponible"})
		return
	}
	ship, err := h.queryService.GetShipByMMSI(mmsi, limit, offset)
	if err != nil {
		if errors.Is(err, ErrShipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno consultando el buque: " + err.Error()})
		return
	}

	// 3. Mapear la entidad de dominio al DTO de Response
	historyResponse := make([]DynamicHistoryResponse, len(ship.Dynamics))
	for i, d := range ship.Dynamics {
		historyResponse[i] = DynamicHistoryResponse{
			Timestamp: d.Timestamp,
			Longitude: d.Longitude,
			Latitude:  d.Latitude,
			Status:    d.Status,
			Speed:     d.Speed,
			Course:    d.Course,
		}
	}

	res := ShipDetailsResponse{
		IMO:        ship.IMO,
		MMSI:       ship.MMSI,
		Callsign:   ship.Callsign,
		Shipname:   ship.Shipname,
		ShipType:   ship.ShipType,
		TotalSpots: len(ship.Dynamics),
		History:    historyResponse,
	}

	// 4. Confirmamos la respuesta status 200
	c.JSON(http.StatusOK, res)
}
