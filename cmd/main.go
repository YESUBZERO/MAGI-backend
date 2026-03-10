package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/YESUBZERO/consumer-service/internal/config"
	"github.com/YESUBZERO/consumer-service/internal/kafka"
	"github.com/YESUBZERO/consumer-service/internal/repository"
)

func main() {
	// Cargar configuración desde variables de entorno
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error cargando la configuración: %v", err)
	}

	// Inicializar la base de datos y repositorio
	db, err := repository.InitDB(cfg.GetDSN())
	if err != nil {
		log.Fatalf("Error Inicializando la base de datos: %v", err)
	}
	repo := repository.NewShipRepository(db)

	// Inicializar productor de Kafka
	producer, err := kafka.NewProducer(cfg.GetKafkaBrokers())
	if err != nil {
		log.Fatalf("Error Inicializando el productor de Kafka: %v", err)
	}
	defer func() {
		if err := producer.Close(); err != nil { // Cerrar el productor al finalizar
			log.Println("Error cerrando el productor de Kafka:", err)
		}
	}()

	// Iniciar el consumidor de Kafka
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()                                       // Detener el contexto al recibir una señal de interrupción
	consumer := kafka.NewConsumer(cfg, producer, repo) // Inicializar el consumidor de Kafka

	log.Println("🚢 [MAGI-CORE] Servicio de Ingesta iniciado... Presiona Ctrl+C para detener.")

	consumer.Start(ctx) // Iniciar el consumo de mensajes

	log.Println("🛑 [MAGI-CORE] Apagando el servicio de forma segura...")

}
