package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"aires-magi/internal/ais"
	"aires-magi/internal/cache"
	"aires-magi/internal/config"
	"aires-magi/internal/database"
	"aires-magi/internal/handlers"
	"aires-magi/internal/middleware"
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
	logger.Info().Msg("🚢 [MAGI-API] Iniciando Microservicio de Consultas HTTP...")

	// 3. Inicializar OpenTelemetry tracing
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otelShutdown, err := observability.InitTracing(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("OpenTelemetry no disponible, continuando sin exportación OTLP")
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = otelShutdown(shutdownCtx)
		}()
	}

	// 4. Inicializar PostgreSQL con pool de conexiones
	db, err := database.InitPostgres(cfg.Database.DSN, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Fallo fatal conectando a PostgreSQL")
	}

	// 5. Inyectar dependencias (Repositorio y Servicios CQRS)
	repo := ais.NewRepository(db)
	queryService := ais.NewQueryService(repo)
	ingestionService := ais.NewIngestionService(repo)
	searchRepo := ais.NewSearchRepository(db)
	searchService := ais.NewShipSearchService(searchRepo)

	// Inicializar ExportService y Worker asíncrono
	exportRepo := ais.NewExportRepository(db)
	exportService := ais.NewExportService(exportRepo, db, "", logger)
	go exportService.StartWorker(ctx)

	// Cliente Redis opcional para diagnósticos
	redisClient, _ := cache.NewRedisClient()

	// Inicializar Middleware de Autenticación Zitadel
	zitadelIssuer := os.Getenv("ZITADEL_ISSUER")
	zitadelJWKSURL := os.Getenv("ZITADEL_JWKS_URL")
	zitadelAudience := os.Getenv("ZITADEL_AUDIENCE")
	if zitadelAudience == "" {
		zitadelAudience = "magi-backend"
	}
	authMiddleware := middleware.NewZitadelAuthMiddleware(middleware.ZitadelConfig{
		Issuer:   zitadelIssuer,
		JWKSURL:  zitadelJWKSURL,
		Audience: zitadelAudience,
	})

	// 6. Router de Gin
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(observability.PrometheusMiddleware())

	// Middleware de CORS
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

	// Middleware de logging con Zerolog
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("duration", time.Since(start)).
			Msg("HTTP Request")
	})

	// Métricas Prometheus (público) y Diagnóstico del Sistema protegido por RBAC (sys:monitor)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	systemHandler := handlers.NewSystemHandler(db, redisClient, &cfg.Kafka)
	r.GET("/api/v1/system/status", authMiddleware.RequireAuth(), authMiddleware.RequireRole("sys:monitor"), systemHandler.GetSystemStatus)

	// 7. Registrar Handlers de AIS y Healthcheck (con RBAC)
	handlers.NewAISHandler(r, queryService, ingestionService, db, logger, authMiddleware)

	// Registrar Handlers de Búsqueda facetada protegidos por RBAC
	searchHandler := handlers.NewSearchHandler(searchService)
	shipsGroup := r.Group("/api/v1/ships")
	shipsGroup.Use(authMiddleware.RequireAuth())
	shipsGroup.Use(authMiddleware.RequireRole("ais:viewer"))
	{
		shipsGroup.GET("", searchHandler.SearchShips)
		shipsGroup.GET("/:mmsi/telemetry", searchHandler.GetShipTelemetry)
	}

	// Registrar Handlers de Exportación Asíncrona protegidos por RBAC (ais:export en todo el grupo)
	exportHandler := handlers.NewExportHandler(exportService, logger)
	exportsGroup := r.Group("/api/v1/exports")
	exportsGroup.Use(authMiddleware.RequireAuth())
	exportsGroup.Use(authMiddleware.RequireRole("ais:export"))
	{
		exportsGroup.POST("", exportHandler.CreateExport)
		exportsGroup.GET("", exportHandler.ListExports)
		exportsGroup.GET("/:id", exportHandler.GetExport)
		exportsGroup.GET("/:id/download", exportHandler.DownloadExport)
	}

	serverAddr := cfg.GetServerAddr()
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: r,
	}

	// 8. Iniciar servidor HTTP en segundo plano
	go func() {
		logger.Info().Str("addr", serverAddr).Msg("🚀 Servidor HTTP escuchando")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Fallo fatal en servidor HTTP")
		}
	}()

	<-ctx.Done()
	logger.Info().Msg("🛑 Señal de terminación recibida. Apagando servidor HTTP ordenadamente...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Error durante apagado forzado del servidor HTTP")
	}

	logger.Info().Msg("✅ Servidor HTTP detenido correctamente.")
}
