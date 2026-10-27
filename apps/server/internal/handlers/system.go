package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"

	"aires-magi/internal/config"
)

type SystemHandler struct {
	db          *gorm.DB
	redisClient *redis.Client
	kafkaCfg    *config.KafkaConfig
}

func NewSystemHandler(db *gorm.DB, rdb *redis.Client, kcfg *config.KafkaConfig) *SystemHandler {
	return &SystemHandler{
		db:          db,
		redisClient: rdb,
		kafkaCfg:    kcfg,
	}
}

func (h *SystemHandler) GetSystemStatus(c *gin.Context) {
	overallStatus := "healthy"
	now := time.Now()

	// 1. Diagnóstico de PostgreSQL
	dbStatus := gin.H{"status": "disconnected"}
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err == nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			if err := sqlDB.PingContext(ctx); err == nil {
				stats := sqlDB.Stats()
				dbStatus = gin.H{
					"status":              "connected",
					"open_connections":    stats.OpenConnections,
					"in_use":              stats.InUse,
					"idle":                stats.Idle,
					"wait_count":          stats.WaitCount,
					"wait_duration":       stats.WaitDuration.String(),
					"max_idle_closed":     stats.MaxIdleClosed,
					"max_lifetime_closed": stats.MaxLifetimeClosed,
				}
			} else {
				dbStatus = gin.H{"status": "unhealthy", "error": err.Error()}
				overallStatus = "degraded"
			}
		}
	}

	// 2. Diagnóstico de Redis
	redisStatus := gin.H{"status": "not_configured"}
	if h.redisClient != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := h.redisClient.Ping(ctx).Err(); err == nil {
			redisStatus = gin.H{"status": "connected"}
		} else {
			redisStatus = gin.H{"status": "unhealthy", "error": err.Error()}
			overallStatus = "degraded"
		}
	}

	// 3. Diagnóstico de Kafka
	kafkaStatus := gin.H{"status": "not_configured"}
	if h.kafkaCfg != nil && len(h.kafkaCfg.Brokers) > 0 {
		conn, err := kafka.DialContext(c.Request.Context(), "tcp", h.kafkaCfg.Brokers[0])
		if err == nil {
			_ = conn.Close()
			kafkaStatus = gin.H{
				"status":  "connected",
				"brokers": h.kafkaCfg.Brokers,
			}
		} else {
			kafkaStatus = gin.H{
				"status":  "unhealthy",
				"brokers": h.kafkaCfg.Brokers,
				"error":   err.Error(),
			}
			overallStatus = "degraded"
		}
	}

	httpCode := http.StatusOK
	if overallStatus == "unhealthy" {
		httpCode = http.StatusServiceUnavailable
	}

	c.JSON(httpCode, gin.H{
		"status":    overallStatus,
		"timestamp": now,
		"database":  dbStatus,
		"redis":     redisStatus,
		"kafka":     kafkaStatus,
	})
}
