package ais

import (
	"net/http"
	"strconv"

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
	Timestamp string  `json:"timestamp"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Status    string  `json:"status"`
	Speed     float64 `json:"speed_knots"`
	Course    float64 `json:"course"`
}

// =========================================================================
// CONTROLADOR / HANDLER
// =========================================================================

// Definimos los atributos del handler
type Handler struct {
	service Service
}

func NewHandler(r *gin.Engine, s Service) {
	h := &Handler{service: s}

	routes := r.Group("/api/v1")
	{
		routes.POST("/static", h.CreateStaticMessage)
		routes.GET("/ship/:imo", h.GetShipByIMO)
	}
}

// CreateStaticMessage maneja la creación/actualizacion via HTTP
func (h *Handler) CreateStaticMessage(c *gin.Context) {
	var req CreateStaticMessageRequest

	// 1. Validar las propiedades del request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos" + err.Error()})
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
	if err := h.service.ProcessStaticMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Datos estáticos del buque procesados correctamente"})

}

func (h *Handler) GetShipByIMO(c *gin.Context) {

	// 1. Validamos el parametro recibido de la ruta
	immoParam := c.Param("imo")
	immo, err := strconv.Atoi(immoParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el IMO debe ser un número válido"})
		return
	}

	// 2. Llamar al servicio GetShipByIMO
	ship, err := h.service.GetShipByIMO(immo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
