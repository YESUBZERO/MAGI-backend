package models

import (
	"time"
)

const (
	ExportStatusPending    = "pending"
	ExportStatusProcessing = "processing"
	ExportStatusCompleted  = "completed"
	ExportStatusFailed     = "failed"

	ExportFormatCSV = "csv"
)

type ExportFilters struct {
	MMSI     *int       `json:"mmsi,omitempty"`
	IMO      *int       `json:"imo,omitempty"`
	ShipType string     `json:"ship_type,omitempty"`
	From     *time.Time `json:"from,omitempty"`
	To       *time.Time `json:"to,omitempty"`
}

type ExportJob struct {
	ID           string     `gorm:"primaryKey;type:varchar(64)" json:"id"`
	UserID       string     `gorm:"type:varchar(128);index;not null" json:"user_id"`
	Status       string     `gorm:"type:varchar(32);index;not null;default:'pending'" json:"status"`
	Format       string     `gorm:"type:varchar(16);not null;default:'csv'" json:"format"`
	Filters      string     `gorm:"type:text" json:"filters"`
	FilePath     string     `gorm:"type:varchar(512)" json:"file_path,omitempty"`
	FileSize     int64      `gorm:"default:0" json:"file_size"`
	TotalRows    int        `gorm:"default:0" json:"total_rows"`
	CreatedAt    time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`
}

func (ExportJob) TableName() string {
	return "export_jobs"
}

type CreateExportRequest struct {
	Format   string     `json:"format" binding:"omitempty,oneof=csv"`
	MMSI     *int       `json:"mmsi,omitempty"`
	IMO      *int       `json:"imo,omitempty"`
	ShipType string     `json:"ship_type,omitempty"`
	From     *time.Time `json:"from,omitempty"`
	To       *time.Time `json:"to,omitempty"`
}

type ExportJobResponse struct {
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	Status       string         `json:"status"`
	Format       string         `json:"format"`
	Filters      *ExportFilters `json:"filters,omitempty"`
	FileSize     int64          `json:"file_size"`
	TotalRows    int            `json:"total_rows"`
	CreatedAt    time.Time      `json:"created_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	DownloadURL  string         `json:"download_url,omitempty"`
}

type ExportJobListResponse struct {
	Total int64               `json:"total"`
	Jobs  []ExportJobResponse `json:"jobs"`
}
