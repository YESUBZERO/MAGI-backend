package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/YESUBZERO/consumer-service/internal/ais"
	"github.com/YESUBZERO/consumer-service/internal/config"
	"github.com/YESUBZERO/consumer-service/internal/database"
	"github.com/YESUBZERO/consumer-service/internal/kafka"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Cargar configuración híbrida (dotenv + envconfig)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error cargando la configuración: %v", err)
	}

	// 2. Inicializar la conexión a la base de datos
	db, err := database.InitPostgres(cfg.GetDSN())
	if err != nil {
		log.Fatalf("Error Inicializando la base de datos: %v", err)
	}

	// 3. Ejecutar AutoMigrate de los modelos exclisivos de GORM
	log.Println("🔄 Ejecutando migraciones de base de datos...")
	if err := db.AutoMigrate(&ais.DBStaticAIS{}, &ais.DBDynamicAIS{}); err != nil {
		log.Fatalf("❌ Error en la automigración de GORM: %v", err)
	}

	// 4. Inyectar el repositorio de buques en el servicio de negocio
	aisRepo := ais.NewRepository(db)
	aisService := ais.NewService(aisRepo)

	// 5. Configurar el contexto para manejo de señales y cierre ordenado
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Iniciar el consumidor de Kafka
	consumer := kafka.NewConsumer(cfg, aisService)
	log.Println("📡 [MAGI-CORE] Iniciando workers concurrentes de Kafka...")
	var consumerWg sync.WaitGroup
	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		consumer.Start(ctx)
	}()

	// 7. Inicializar el Servidor HTTP de Gin para consultas
	r := gin.Default()

	// Endpoint de salud profundo para monitoreo y Docker HEALTHCHECK (valida conectividad con BD)
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
			"database": "connected",
		})
	})

	ais.NewHandler(r, aisService, aisService) // Registramos las rutas del handler de AIS (consultas e ingesta)

	serverPort := cfg.GetServerPort()
	// Arrancamos el servidor HTTP en una goroutine para gestionar el cierre ordenado
	srv := &http.Server{
		Addr:    serverPort,
		Handler: r,
	}

	go func() {
		log.Printf("🚀 [MAGI-CORE] Servidor HTTP escuchando en el puerto %s", serverPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Error en el servidor HTTP: %v", err)
		}
	}()

	// 8. Esperar a recibir una señal de interrupción para cerrar ordenadamente
	log.Println("🚢 [MAGI-CORE] Todos los servicios en marcha. Presiona Ctrl+C para detener.")
	<-ctx.Done()

	// 9. Iniciar el proceso de cierre ordenado de los servicios
	log.Println("🛑 [MAGI-CORE] Apagando el servicio de forma segura...")

	// Configurar margen de tiempo para que los procesos en curso terminen
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ Error durante el cierre del servidor HTTP: %v", err)
	}

	// Esperar que los workers y el consumidor de Kafka finalicen limpiamente
	consumerWg.Wait()

	log.Println("✅ [MAGI-CORE] Sistema completamente apagado.")
}
