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

	// Ejecutar AutoMigrate para tablas AIS y ExportJobs
	if err := db.AutoMigrate(&models.DBStaticAIS{}, &models.DBDynamicAIS{}, &models.ExportJob{}); err != nil {
		return nil, fmt.Errorf("error en automigración AIS: %w", err)
	}

	// Optimización P1: Extensión pg_trgm e índice GIN Trigram para acelerar búsquedas de nombres de buques
	if db.Dialector.Name() == "postgres" {
		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
			zlog.Warn().Err(err).Msg("No se pudo crear extensión pg_trgm (requiere permisos en PostgreSQL)")
		} else {
			if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_static_ais_shipname_trgm ON static_ais USING gin (LOWER(shipname) gin_trgm_ops)").Error; err != nil {
				zlog.Warn().Err(err).Msg("No se pudo crear índice GIN trigram en shipname")
			} else {
				zlog.Info().Msg("Índice GIN trigram verificado en static_ais(LOWER(shipname))")
			}
		}
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
