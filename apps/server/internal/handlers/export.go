package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"aires-magi/internal/ais"
	"aires-magi/internal/middleware"
	"aires-magi/internal/models"
)

type ExportHandler struct {
	service ais.ExportService
	logger  zerolog.Logger
}

func NewExportHandler(svc ais.ExportService, logger zerolog.Logger) *ExportHandler {
	return &ExportHandler{
		service: svc,
		logger:  logger.With().Str("component", "export-handler").Logger(),
	}
}

func (h *ExportHandler) CreateExport(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no autenticado"})
		return
	}

	var req models.CreateExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "solicitud inválida: " + err.Error()})
		return
	}

	job, err := h.service.CreateExport(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Error creando job de exportación")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error encolando exportación"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Trabajo de exportación encolado exitosamente",
		"job_id":  job.ID,
		"status":  job.Status,
	})
}

func (h *ExportHandler) ListExports(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no autenticado"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	jobs, total, err := h.service.ListExports(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]models.ExportJobResponse, len(jobs))
	for i, j := range jobs {
		var f *models.ExportFilters
		if j.Filters != "" {
			var flt models.ExportFilters
			_ = json.Unmarshal([]byte(j.Filters), &flt)
			f = &flt
		}
		resp[i] = models.ExportJobResponse{
			ID:           j.ID,
			UserID:       j.UserID,
			Status:       j.Status,
			Format:       j.Format,
			Filters:      f,
			FileSize:     j.FileSize,
			TotalRows:    j.TotalRows,
			CreatedAt:    j.CreatedAt,
			CompletedAt:  j.CompletedAt,
			ErrorMessage: j.ErrorMessage,
			DownloadURL:  fmt.Sprintf("/api/v1/exports/%s/download", j.ID),
		}
	}

	c.JSON(http.StatusOK, models.ExportJobListResponse{
		Total: total,
		Jobs:  resp,
	})
}

func (h *ExportHandler) GetExport(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no autenticado"})
		return
	}

	jobID := c.Param("id")
	job, err := h.service.GetExport(c.Request.Context(), userID, jobID)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			c.JSON(http.StatusForbidden, gin.H{"error": "acceso denegado al job solicitado"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "job de exportación no encontrado"})
		return
	}

	var f *models.ExportFilters
	if job.Filters != "" {
		var flt models.ExportFilters
		_ = json.Unmarshal([]byte(job.Filters), &flt)
		f = &flt
	}

	c.JSON(http.StatusOK, models.ExportJobResponse{
		ID:           job.ID,
		UserID:       job.UserID,
		Status:       job.Status,
		Format:       job.Format,
		Filters:      f,
		FileSize:     job.FileSize,
		TotalRows:    job.TotalRows,
		CreatedAt:    job.CreatedAt,
		CompletedAt:  job.CompletedAt,
		ErrorMessage: job.ErrorMessage,
		DownloadURL:  fmt.Sprintf("/api/v1/exports/%s/download", job.ID),
	})
}

func (h *ExportHandler) DownloadExport(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no autenticado"})
		return
	}

	jobID := c.Param("id")
	job, err := h.service.GetExport(c.Request.Context(), userID, jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job de exportación no encontrado"})
		return
	}

	if job.Status != models.ExportStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "el archivo no está listo para descarga",
			"status": job.Status,
		})
		return
	}

	exportDir := h.service.GetExportDir()
	if exportDir == "" {
		exportDir = filepath.Join(os.TempDir(), "magi_exports")
	}
	cleanExportDir := filepath.Clean(exportDir)
	cleanFilePath := filepath.Clean(job.FilePath)

	rel, err := filepath.Rel(cleanExportDir, cleanFilePath)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." || filepath.IsAbs(rel) {
		c.JSON(http.StatusForbidden, gin.H{"error": "ruta de archivo fuera del directorio de exportación permitido"})
		return
	}

	if _, err := os.Stat(cleanFilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "el archivo físico no existe en disco"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"export_%s.csv.gz\"", job.ID))
	c.Header("Content-Type", "application/gzip")
	c.File(cleanFilePath)
}
