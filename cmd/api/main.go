package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YESUBZERO/consumer-service/internal/ais"
	"github.com/YESUBZERO/consumer-service/internal/config"
	"github.com/YESUBZERO/consumer-service/internal/database"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("🚢 [MAGI-API] Iniciando Microservicio de Consultas HTTP (Read Path)...")

	// 1. Cargar configuración exclusiva de la API (DATABASE_DSN y PORT, sin dependencias de Kafka)
	cfg, err := config.LoadAPI()
	if err != nil {
		log.Fatalf("❌ [MAGI-API] Error cargando la configuración: %v", err)
	}

	// 2. Inicializar la conexión a PostgreSQL con pool optimizado
	db, err := database.InitPostgres(cfg.GetDSN())
	if err != nil {
		log.Fatalf("❌ [MAGI-API] Error conectando a la base de datos: %v", err)
	}

	// 3. Ejecutar AutoMigrate para asegurar que las tablas e índices existan
	log.Println("🔄 [MAGI-API] Verificando esquema de base de datos...")
	if err := db.AutoMigrate(&ais.DBStaticAIS{}, &ais.DBDynamicAIS{}); err != nil {
		log.Fatalf("❌ [MAGI-API] Error en la migración de base de datos: %v", err)
	}

	// 4. Instanciar el repositorio y el servicio exclusivo de consultas (QueryService)
	aisRepo := ais.NewRepository(db)
	queryService := ais.NewQueryService(aisRepo)

	// 5. Configurar el contexto de cancelación para apagado ordenado (Graceful Shutdown)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Inicializar el router de Gin
	r := gin.Default()

	// Endpoint de salud profundo para monitoreo, Kubernetes y Docker HEALTHCHECK
	r.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "error al acceder a la instancia de base de datos",
			})
			return
		}

		pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(pingCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "base de datos no responde al ping",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "magi-api",
			"database": "connected",
		})
	})

	// 7. Registrar las rutas HTTP de consulta de telemetría y buques
	ais.NewHandler(r, queryService)

	serverPort := cfg.GetServerPort()
	srv := &http.Server{
		Addr:    serverPort,
		Handler: r,
	}

	// 8. Iniciar el servidor HTTP en segundo plano
	go func() {
		log.Printf("🚀 [MAGI-API] Servidor HTTP escuchando en %s", serverPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ [MAGI-API] Error en el servidor HTTP: %v", err)
		}
	}()

	log.Println("🚢 [MAGI-API] Microservicio de consultas en marcha. Presiona Ctrl+C para detener.")
	<-ctx.Done()

	// 9. Apagado ordenado del servidor HTTP
	log.Println("🛑 [MAGI-API] Señal de terminación recibida. Apagando servidor HTTP de forma segura...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ [MAGI-API] Error durante el cierre del servidor HTTP: %v", err)
	}

	log.Println("✅ [MAGI-API] Servidor HTTP finalizado exitosamente.")
}
