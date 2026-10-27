package ais

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"aires-magi/internal/models"
)

type ExportService interface {
	CreateExport(ctx context.Context, userID string, req models.CreateExportRequest) (*models.ExportJob, error)
	GetExport(ctx context.Context, userID, jobID string) (*models.ExportJob, error)
	ListExports(ctx context.Context, userID string, limit, offset int) ([]models.ExportJob, int64, error)
	StartWorker(ctx context.Context)
	ProcessJob(ctx context.Context, job *models.ExportJob) error
	GetExportDir() string
}

type exportService struct {
	repo       ExportRepository
	db         *gorm.DB
	exportDir  string
	logger     zerolog.Logger
	jobTrigger chan struct{}
	sem        chan struct{}
}

func NewExportService(repo ExportRepository, db *gorm.DB, exportDir string, logger zerolog.Logger) ExportService {
	if exportDir == "" {
		exportDir = filepath.Join(os.TempDir(), "magi_exports")
	}
	_ = os.MkdirAll(exportDir, 0755)

	return &exportService{
		repo:       repo,
		db:         db,
		exportDir:  exportDir,
		logger:     logger.With().Str("component", "export-service").Logger(),
		jobTrigger: make(chan struct{}, 100),
		sem:        make(chan struct{}, 5),
	}
}

func (s *exportService) CreateExport(ctx context.Context, userID string, req models.CreateExportRequest) (*models.ExportJob, error) {
	filterBytes, err := json.Marshal(models.ExportFilters{
		MMSI:     req.MMSI,
		IMO:      req.IMO,
		ShipType: req.ShipType,
		From:     req.From,
		To:       req.To,
	})
	if err != nil {
		return nil, fmt.Errorf("error serializando filtros: %w", err)
	}

	job := &models.ExportJob{
		ID:        uuid.New().String(),
		UserID:    userID,
		Status:    models.ExportStatusPending,
		Format:    models.ExportFormatCSV,
		Filters:   string(filterBytes),
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("error creando job de exportación: %w", err)
	}

	select {
	case s.jobTrigger <- struct{}{}:
	default:
	}

	return job, nil
}

func (s *exportService) GetExport(ctx context.Context, userID, jobID string) (*models.ExportJob, error) {
	job, err := s.repo.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.UserID != userID {
		return nil, os.ErrPermission
	}
	return job, nil
}

func (s *exportService) ListExports(ctx context.Context, userID string, limit, offset int) ([]models.ExportJob, int64, error) {
	return s.repo.ListJobsByUserID(ctx, userID, limit, offset)
}

func (s *exportService) GetExportDir() string {
	return s.exportDir
}

func (s *exportService) StartWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	s.logger.Info().Msg("Worker asíncrono de exportación .csv.gz iniciado")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("Deteniendo worker de exportación...")
			return
		case <-ticker.C:
			s.processPendingJobs(ctx)
		case <-s.jobTrigger:
			s.processPendingJobs(ctx)
		}
	}
}

func (s *exportService) processPendingJobs(ctx context.Context) {
	// Optimización P1: Control de concurrencia mediante canal semáforo
	slots := cap(s.sem) - len(s.sem)
	if slots <= 0 {
		return
	}

	jobs, err := s.repo.ClaimPendingJobs(ctx, slots)
	if err != nil {
		s.logger.Error().Err(err).Msg("Error obteniendo jobs pendientes")
		return
	}

	for _, job := range jobs {
		currentJob := job
		select {
		case s.sem <- struct{}{}:
			go func() {
				defer func() { <-s.sem }()
				if err := s.ProcessJob(ctx, &currentJob); err != nil {
					s.logger.Error().Err(err).Str("job_id", currentJob.ID).Msg("Fallo procesando job de exportación")
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}

func (s *exportService) ProcessJob(ctx context.Context, job *models.ExportJob) (err error) {
	var filters models.ExportFilters
	if job.Filters != "" {
		_ = json.Unmarshal([]byte(job.Filters), &filters)
	}

	fileName := fmt.Sprintf("export_%s.csv.gz", job.ID)
	filePath := filepath.Join(s.exportDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		_ = s.repo.UpdateJobStatus(ctx, job.ID, models.ExportStatusFailed, "", 0, 0, err.Error())
		return err
	}

	success := false
	defer func() {
		_ = file.Close()
		if !success {
			_ = os.Remove(filePath)
		}
	}()

	// Optimización P1: Buffer de E/S de 64KB antes de compresión gzip
	bufFile := bufio.NewWriterSize(file, 64*1024)
	gzWriter := gzip.NewWriter(bufFile)
	defer gzWriter.Close()

	csvWriter := csv.NewWriter(gzWriter)
	defer csvWriter.Flush()

	headers := []string{"timestamp", "mmsi", "imo", "shipname", "latitude", "longitude", "speed", "course", "status"}
	if err := csvWriter.Write(headers); err != nil {
		_ = s.repo.UpdateJobStatus(ctx, job.ID, models.ExportStatusFailed, "", 0, 0, err.Error())
		return err
	}

	query := s.db.WithContext(ctx).Table("dynamic_ais d").
		Select("d.timestamp, d.mmsi, COALESCE(s.imo, 0) as imo, COALESCE(s.shipname, 'UNKNOWN') as shipname, d.latitude, d.longitude, d.speed, d.course, d.status").
		Joins("LEFT JOIN static_ais s ON d.mmsi = s.mmsi")

	if filters.MMSI != nil {
		query = query.Where("d.mmsi = ?", *filters.MMSI)
	}
	if filters.IMO != nil {
		query = query.Where("s.imo = ?", *filters.IMO)
	}
	if filters.ShipType != "" {
		query = query.Where("LOWER(s.ship_type) = LOWER(?)", filters.ShipType)
	}
	if filters.From != nil {
		query = query.Where("d.timestamp >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("d.timestamp <= ?", *filters.To)
	}

	rows, err := query.Order("d.timestamp ASC").Rows()
	if err != nil {
		_ = s.repo.UpdateJobStatus(ctx, job.ID, models.ExportStatusFailed, "", 0, 0, err.Error())
		return err
	}
	defer rows.Close()

	totalRows := 0
	// Optimización P1: Prealocación de slice fuera del bucle para eliminar presión sobre el GC
	record := make([]string, 9)
	for rows.Next() {
		var (
			ts       time.Time
			mmsi     int
			imo      int
			shipname string
			lat      float64
			lon      float64
			speed    float64
			course   float64
			status   string
		)

		if err := rows.Scan(&ts, &mmsi, &imo, &shipname, &lat, &lon, &speed, &course, &status); err != nil {
			_ = s.repo.UpdateJobStatus(ctx, job.ID, models.ExportStatusFailed, "", 0, 0, err.Error())
			return err
		}

		record[0] = ts.Format(time.RFC3339)
		record[1] = strconv.Itoa(mmsi)
		record[2] = strconv.Itoa(imo)
		record[3] = shipname
		record[4] = strconv.FormatFloat(lat, 'f', 6, 64)
		record[5] = strconv.FormatFloat(lon, 'f', 6, 64)
		record[6] = strconv.FormatFloat(speed, 'f', 2, 64)
		record[7] = strconv.FormatFloat(course, 'f', 2, 64)
		record[8] = status

		if err := csvWriter.Write(record); err != nil {
			_ = s.repo.UpdateJobStatus(ctx, job.ID, models.ExportStatusFailed, "", 0, 0, err.Error())
			return err
		}
		totalRows++
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		_ = s.repo.UpdateJobStatus(ctx, job.ID, models.ExportStatusFailed, "", 0, 0, err.Error())
		return err
	}
	if err := gzWriter.Close(); err != nil {
		_ = s.repo.UpdateJobStatus(ctx, job.ID, models.ExportStatusFailed, "", 0, 0, err.Error())
		return err
	}
	if err := bufFile.Flush(); err != nil {
		_ = s.repo.UpdateJobStatus(ctx, job.ID, models.ExportStatusFailed, "", 0, 0, err.Error())
		return err
	}

	fi, err := file.Stat()
	fileSize := int64(0)
	if err == nil {
		fileSize = fi.Size()
	}

	if err = s.repo.UpdateJobStatus(ctx, job.ID, models.ExportStatusCompleted, filePath, fileSize, totalRows, ""); err != nil {
		return err
	}

	success = true
	return nil
}
