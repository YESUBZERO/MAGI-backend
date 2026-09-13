package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/YESUBZERO/consumer-service/internal/ais"
	"github.com/YESUBZERO/consumer-service/internal/config"
	"github.com/YESUBZERO/consumer-service/internal/database"
	"github.com/YESUBZERO/consumer-service/internal/kafka"
)

func main() {
	log.Println("📡 [MAGI-WORKER] Iniciando Microservicio Ingestor de Kafka (Write Path)...")

	// 1. Cargar configuración de Kafka y base de datos
	cfg, err := config.LoadWorker()
	if err != nil {
		log.Fatalf("❌ [MAGI-WORKER] Error cargando la configuración: %v", err)
	}

	// 2. Inicializar la conexión a PostgreSQL con pool optimizado para inserciones
	db, err := database.InitPostgres(cfg.GetDSN())
	if err != nil {
		log.Fatalf("❌ [MAGI-WORKER] Error conectando a la base de datos: %v", err)
	}

	// 3. Ejecutar AutoMigrate de esquemas e índices únicos
	log.Println("🔄 [MAGI-WORKER] Verificando esquema de base de datos...")
	if err := db.AutoMigrate(&ais.DBStaticAIS{}, &ais.DBDynamicAIS{}); err != nil {
		log.Fatalf("❌ [MAGI-WORKER] Error en la migración de base de datos: %v", err)
	}

	// 4. Instanciar el repositorio y el servicio exclusivo de ingesta (IngestionService)
	aisRepo := ais.NewRepository(db)
	ingestionService := ais.NewIngestionService(aisRepo)

	// 5. Configurar el contexto de cancelación para apagado ordenado (Graceful Shutdown)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Instanciar e iniciar el consumidor de Kafka con el worker pool concurrente
	consumer := kafka.NewConsumer(cfg, ingestionService)
	staticTopic, dynamicTopic := cfg.GetKafkaTopics()
	log.Printf("📡 [MAGI-WORKER] Consumer Group '%s' escuchando tópicos: [%s, %s]",
		cfg.GetKafkaGroupID(), staticTopic, dynamicTopic)

	var consumerWg sync.WaitGroup
	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		consumer.Start(ctx)
	}()

	log.Println("🚢 [MAGI-WORKER] Ingestor de telemetría en marcha. Presiona Ctrl+C para detener.")
	<-ctx.Done()

	// 7. Apagado ordenado: esperar que todos los workers finalicen y confirmen offsets pendientes
	log.Println("🛑 [MAGI-WORKER] Señal de terminación recibida. Drenando workers de Kafka de forma segura...")
	consumerWg.Wait()

	log.Println("✅ [MAGI-WORKER] Ingestor de Kafka finalizado limpiamente.")
}
