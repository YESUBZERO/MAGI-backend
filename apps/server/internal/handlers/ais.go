package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"aires-magi/internal/ais"
	"aires-magi/internal/models"
)

// AISHandler gestiona los endpoints HTTP de buques y telemetría AIS
type AISHandler struct {
	queryService     ais.QueryService
	ingestionService ais.IngestionService
	db               *gorm.DB
	logger           zerolog.Logger
}

// NewAISHandler construye y registra las rutas AIS en el router de Gin
func NewAISHandler(
	r *gin.Engine,
	q ais.QueryService,
	ing ais.IngestionService,
	db *gorm.DB,
	log zerolog.Logger,
) *AISHandler {
	h := &AISHandler{
		queryService:     q,
		ingestionService: ing,
		db:               db,
		logger:           log.With().Str("component", "ais-handler").Logger(),
	}

	// Health check profundo (verifica ping a base de datos)
	r.GET("/health", h.HealthCheck)

	// Grupo de rutas de la API v1
	v1 := r.Group("/api/v1")
	{
		if h.ingestionService != nil {
			v1.POST("/static", h.CreateStaticMessage)
		}
		if h.queryService != nil {
			v1.GET("/ship/:imo", h.GetShipByIMO)
			v1.GET("/ship/mmsi/:mmsi", h.GetShipByMMSI)
		}
	}

	return h
}

// HealthCheck valida el estado del servicio y la conectividad a la base de datos
func (h *AISHandler) HealthCheck(c *gin.Context) {
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			h.logger.Error().Err(err).Msg("Error accediendo a la instancia SQL para healthcheck")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "error al acceder a la instancia de base de datos",
			})
			return
		}

		pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(pingCtx); err != nil {
			h.logger.Error().Err(err).Msg("Base de datos no respondió al ping del healthcheck")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "base de datos no responde al ping",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"service":  "aires-magi-server",
		"database": "connected",
	})
}

// CreateStaticMessage maneja la creación o actualización de buques vía HTTP POST
func (h *AISHandler) CreateStaticMessage(c *gin.Context) {
	var req models.CreateStaticMessageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Datos de entrada inválidos para CreateStaticMessage")
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return
	}

	msg := &models.StaticAIS{
		MsgType:  req.MsgType,
		IMO:      req.IMO,
		MMSI:     req.MMSI,
		Callsign: req.Callsign,
		Shipname: req.Shipname,
		ShipType: req.ShipType,
	}

	if h.ingestionService == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "módulo de ingesta no disponible"})
		return
	}

	if err := h.ingestionService.ProcessStaticMessage(msg); err != nil {
		h.logger.Error().Err(err).Int("mmsi", req.MMSI).Msg("Error procesando mensaje estático")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info().Int("mmsi", req.MMSI).Str("shipname", req.Shipname).Msg("Buque registrado exitosamente")
	c.JSON(http.StatusCreated, gin.H{"message": "Datos estáticos del buque procesados correctamente"})
}

// GetShipByIMO busca un buque por su número IMO
func (h *AISHandler) GetShipByIMO(c *gin.Context) {
	imoParam := c.Param("imo")
	imo, err := strconv.Atoi(imoParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el IMO debe ser un número válido"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if h.queryService == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "módulo de consultas no disponible"})
		return
	}

	ship, err := h.queryService.GetShipByIMO(imo, limit, offset)
	if err != nil {
		if errors.Is(err, models.ErrShipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error().Err(err).Int("imo", imo).Msg("Error consultando buque por IMO")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno consultando el buque: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, mapToResponse(ship))
}

// GetShipByMMSI busca un buque por su código MMSI
func (h *AISHandler) GetShipByMMSI(c *gin.Context) {
	mmsiParam := c.Param("mmsi")
	mmsi, err := strconv.Atoi(mmsiParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el MMSI debe ser un número válido"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if h.queryService == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "módulo de consultas no disponible"})
		return
	}

	ship, err := h.queryService.GetShipByMMSI(mmsi, limit, offset)
	if err != nil {
		if errors.Is(err, models.ErrShipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error().Err(err).Int("mmsi", mmsi).Msg("Error consultando buque por MMSI")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno consultando el buque: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, mapToResponse(ship))
}

func mapToResponse(ship *models.StaticAIS) models.ShipDetailsResponse {
	history := make([]models.DynamicHistoryResponse, len(ship.Dynamics))
	for i, d := range ship.Dynamics {
		history[i] = models.DynamicHistoryResponse{
			Timestamp: d.Timestamp,
			Longitude: d.Longitude,
			Latitude:  d.Latitude,
			Status:    d.Status,
			Speed:     d.Speed,
			Course:    d.Course,
		}
	}

	return models.ShipDetailsResponse{
		IMO:        ship.IMO,
		MMSI:       ship.MMSI,
		Callsign:   ship.Callsign,
		Shipname:   ship.Shipname,
		ShipType:   ship.ShipType,
		TotalSpots: len(ship.Dynamics),
		History:    history,
	}
}
