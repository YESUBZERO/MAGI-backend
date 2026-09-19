package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"aires-magi/internal/ais"
	"aires-magi/internal/config"
	"aires-magi/internal/database"
	"aires-magi/internal/handlers"
	"aires-magi/internal/messaging"
	"aires-magi/internal/observability"
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
	default:
		zlog = zlog.Level(zerolog.InfoLevel)
	}

	return zlog
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("Error cargando configuración: " + err.Error())
	}

	logger := setupLogger(cfg.Log.Level)
	logger.Info().Msg("🚢 [MAGI-ALL] Iniciando API y Worker de forma unificada...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otelShutdown, err := observability.InitTracing(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("OpenTelemetry no disponible, continuando sin exportador OTLP")
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = otelShutdown(shutdownCtx)
		}()
	}

	db, err := database.InitPostgres(cfg.Database.DSN, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Fallo fatal conectando a PostgreSQL")
	}

	repo := ais.NewRepository(db)
	service := ais.NewService(repo)

	// Iniciar consumidor de Kafka en segundo plano
	go func() {
		consumer := messaging.NewConsumer(&cfg.Kafka, service, logger)
		consumer.Start(ctx)
	}()

	// Servidor HTTP Gin
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	corsOrigin := cfg.Server.CorsOrigin
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	handlers.NewAISHandler(r, service, service, db, logger)

	serverAddr := cfg.GetServerAddr()
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: r,
	}

	go func() {
		logger.Info().Str("addr", serverAddr).Msg("🚀 Servidor HTTP unificado escuchando")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Fallo fatal en servidor HTTP")
		}
	}()

	<-ctx.Done()
	logger.Info().Msg("🛑 Apagando servicios...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Info().Msg("✅ Sistema detenido.")
}
