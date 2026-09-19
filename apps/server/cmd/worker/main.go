package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"aires-magi/internal/ais"
	"aires-magi/internal/config"
	"aires-magi/internal/database"
	"aires-magi/internal/messaging"
)

func setupLogger(level string) zerolog.Logger {
	var zlog zerolog.Logger
	if os.Getenv("ENV") == "production" || os.Getenv("LOG_FORMAT") == "json" {
		zlog = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		zlog = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Timestamp().Logger()
	}

	switch level {
	case "debug":
		zlog = zlog.Level(zerolog.DebugLevel)
	case "info":
		zlog = zlog.Level(zerolog.InfoLevel)
	case "warn":
		zlog = zlog.Level(zerolog.WarnLevel)
	case "error":
		zlog = zlog.Level(zerolog.ErrorLevel)
	default:
		zlog = zlog.Level(zerolog.InfoLevel)
	}

	return zlog
}

func main() {
	// 1. Cargar configuración con Viper
	cfg, err := config.Load()
	if err != nil {
		panic("Error cargando configuración: " + err.Error())
	}

	// 2. Inicializar Zerolog
	logger := setupLogger(cfg.Log.Level)
	logger.Info().Msg("📡 [MAGI-WORKER] Iniciando Microservicio Ingestor de Kafka (Write Path)...")

	// 3. Inicializar PostgreSQL con pool de conexiones
	db, err := database.InitPostgres(cfg.Database.DSN, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Fallo fatal conectando a PostgreSQL")
	}

	// 4. Inyectar dependencias (Repositorio y Servicio de Ingesta)
	repo := ais.NewRepository(db)
	ingestionService := ais.NewIngestionService(repo)

	// 5. Configurar manejo de señales
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Iniciar consumidor de Kafka con worker pool concurrente
	consumer := messaging.NewConsumer(&cfg.Kafka, ingestionService, logger)

	logger.Info().
		Str("group", cfg.Kafka.GroupID).
		Strs("brokers", cfg.Kafka.Brokers).
		Msg("Workers de Kafka en marcha. Presiona Ctrl+C para detener.")

	consumer.Start(ctx)

	logger.Info().Msg("✅ [MAGI-WORKER] Ingestor de Kafka finalizado limpiamente.")
}
