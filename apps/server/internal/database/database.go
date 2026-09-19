package database

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"aires-magi/internal/models"
)

// InitPostgres inicializa la conexión a PostgreSQL con pool de conexiones y automigraciones
func InitPostgres(dsn string, zlog zerolog.Logger) (*gorm.DB, error) {
	newLogger := logger.New(
		&zerologWriter{logger: zlog},
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  false,
		},
	)

	var db *gorm.DB
	var err error

	if dsn == "" || dsn == "sqlite" {
		db, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{Logger: newLogger})
	} else {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: newLogger})
	}

	if err != nil {
		return nil, fmt.Errorf("error conectando a la base de datos: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo instancia SQL nativa: %w", err)
	}

	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Ejecutar AutoMigrate para tablas AIS
	if err := db.AutoMigrate(&models.DBStaticAIS{}, &models.DBDynamicAIS{}); err != nil {
		return nil, fmt.Errorf("error en automigración AIS: %w", err)
	}

	zlog.Info().Msg("Conexión a base de datos PostgreSQL establecida y esquemas verificados")
	return db, nil
}

type zerologWriter struct {
	logger zerolog.Logger
}

func (w *zerologWriter) Printf(format string, v ...interface{}) {
	w.logger.Debug().Msgf(format, v...)
}
