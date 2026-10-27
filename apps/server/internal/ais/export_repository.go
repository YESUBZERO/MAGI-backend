package ais

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aires-magi/internal/models"
)

type ExportRepository interface {
	CreateJob(ctx context.Context, job *models.ExportJob) error
	GetJobByID(ctx context.Context, id string) (*models.ExportJob, error)
	ListJobsByUserID(ctx context.Context, userID string, limit, offset int) ([]models.ExportJob, int64, error)
	UpdateJobStatus(ctx context.Context, id string, status string, filePath string, fileSize int64, totalRows int, errMsg string) error
	GetPendingJobs(ctx context.Context, limit int) ([]models.ExportJob, error)
	ClaimPendingJobs(ctx context.Context, limit int) ([]models.ExportJob, error)
}

type exportRepository struct {
	db *gorm.DB
}

func NewExportRepository(db *gorm.DB) ExportRepository {
	return &exportRepository{db: db}
}

func (r *exportRepository) CreateJob(ctx context.Context, job *models.ExportJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *exportRepository) GetJobByID(ctx context.Context, id string) (*models.ExportJob, error) {
	var job models.ExportJob
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *exportRepository) ListJobsByUserID(ctx context.Context, userID string, limit, offset int) ([]models.ExportJob, int64, error) {
	var jobs []models.ExportJob
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ExportJob{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

func (r *exportRepository) UpdateJobStatus(ctx context.Context, id string, status string, filePath string, fileSize int64, totalRows int, errMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if filePath != "" {
		updates["file_path"] = filePath
	}
	if fileSize > 0 {
		updates["file_size"] = fileSize
	}
	if totalRows > 0 {
		updates["total_rows"] = totalRows
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	if status == models.ExportStatusCompleted || status == models.ExportStatusFailed {
		now := time.Now()
		updates["completed_at"] = &now
	}

	return r.db.WithContext(ctx).Model(&models.ExportJob{}).Where("id = ?", id).Updates(updates).Error
}

func (r *exportRepository) GetPendingJobs(ctx context.Context, limit int) ([]models.ExportJob, error) {
	if limit <= 0 {
		limit = 10
	}
	var jobs []models.ExportJob
	err := r.db.WithContext(ctx).
		Where("status = ?", models.ExportStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error
	return jobs, err
}

func (r *exportRepository) ClaimPendingJobs(ctx context.Context, limit int) ([]models.ExportJob, error) {
	if limit <= 0 {
		limit = 10
	}
	var claimedJobs []models.ExportJob
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&models.ExportJob{}).
			Where("status = ?", models.ExportStatusPending).
			Order("created_at ASC").
			Limit(limit)

		if tx.Dialector.Name() == "postgres" {
			query = query.Clauses(clause.Locking{
				Strength: "UPDATE",
				Options:  "SKIP LOCKED",
			})
		}

		if err := query.Find(&claimedJobs).Error; err != nil {
			return err
		}

		if len(claimedJobs) == 0 {
			return nil
		}

		ids := make([]string, len(claimedJobs))
		for i, job := range claimedJobs {
			ids[i] = job.ID
		}

		if err := tx.Model(&models.ExportJob{}).
			Where("id IN ?", ids).
			Update("status", models.ExportStatusProcessing).Error; err != nil {
			return err
		}

		for i := range claimedJobs {
			claimedJobs[i].Status = models.ExportStatusProcessing
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return claimedJobs, nil
}
