package kafka

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/IBM/sarama"
	"github.com/YESUBZERO/consumer-service/internal/models"
)

// WorkerPool define la cantidad de workers concurrentes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	WorkerPool  = 5
)

// KafkaConfigReader define la interface para la configuración de Kafka
type KafkaConfigReader interface {
	GetKafkaBrokers() []string
	GetKafkaTopics() (string, string, string, string)
}

// ShipStorage define la interface para la base de datos
type ShipStorer interface {
	IMOExists(imo int) (bool, error)
	MMSIExists(mmsi int) (bool, error)
	SaveShip(ship *models.Ship) error
	SavePosition(position *models.ShipPosition) error
}

// MessageSender define la interface para el productor de Kafka
type MessageSender interface {
	SendMessage(topic, message string)
}

// Consumer define la interface para el consumidor de Kafka
type Consumer struct {
	cfg      KafkaConfigReader
	producer *Producer
	repo     ShipStorer
}

// NewConsumer crea un nuevo consumidor de Kafka
func NewConsumer(cr KafkaConfigReader, p *Producer, r ShipStorer) *Consumer {
	return &Consumer{
		cfg:      cr,
		producer: p,
		repo:     r,
	}
}

// ConsumeMessages consume los mensajes de Kafka
func (c *Consumer) Start(ctx context.Context) {

	brokers := c.cfg.GetKafkaBrokers()
	staticTopic, dynamicTopic, _, _ := c.cfg.GetKafkaTopics()

	consumer, err := sarama.NewConsumer(brokers, nil)
	if err != nil {
		log.Fatalf("❌ Error creando consumidor de Kafka: %v", err)
	}
	defer consumer.Close()

	messageChannel := make(chan *sarama.ConsumerMessage, 500) // Buffer para mensajes
	var wg sync.WaitGroup

	// 🔄 Iniciar pool de workers
	for i := 0; i < WorkerPool; i++ {
		wg.Add(1)
		go c.worker(i, &wg, messageChannel, staticTopic, dynamicTopic)
	}

	// 🔗 Suscripción a tópicos
	topics := []string{staticTopic, dynamicTopic}
	log.Printf("📡 Escuchando tópicos: %s, %s", staticTopic, dynamicTopic)
	for _, topic := range topics {

		pc, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
		if err != nil {
			log.Fatalf("❌ Error en tópico %s: %v", topic, err)
			continue
		}

		go func(pc sarama.PartitionConsumer) {
			defer pc.Close()
			for msg := range pc.Messages() {
				messageChannel <- msg
			}
		}(pc)
	}
	<-ctx.Done()          // Esperar a que el contexto se cancele
	close(messageChannel) // Cerrar el canal de mensajes
	wg.Wait()             // Esperar a que todos los workers terminen
}

// Worker maneja la lógica de un worker
func (c *Consumer) worker(id int, wg *sync.WaitGroup, ch <-chan *sarama.ConsumerMessage, static, dynamic string) {
	defer wg.Done()

	for msg := range ch {
		switch msg.Topic {
		case static:
			var ship models.Ship
			json.Unmarshal(msg.Value, &ship)
			c.processStatic(id, ship)
		case dynamic:
			var sp models.ShipPosition
			json.Unmarshal(msg.Value, &sp)
			c.processDynamic(id, sp)
		}
	}
}

// Procesa mensajes estaticos y envia a scraper
func (c *Consumer) processStatic(id int, ship models.Ship) {
	if ship.IMO == 0 || ship.MMSI == 0 {
		return
	}
	// Verificar si el barco ya existe
	exists, err := c.repo.IMOExists(ship.IMO)
	if err != nil {
		log.Printf("❌ [Worker %d] Error DB: %v", id, err)
	}
	// Si el barco no existe, se almacena
	if !exists {
		log.Printf("%s[Worker %d]%s 🚢 Registrando nuevo buque (IMO: %d / Name: %s)", colorBlue, id, colorReset, ship.IMO, ship.Shipname)
		c.repo.SaveShip(&ship)
	}
}

// processEnriched procesa mensajes enriquecidos y los guarda en la base de datos
func (c *Consumer) processDynamic(id int, position models.ShipPosition) {
	// Verificar si el barco ya existe
	exists, err := c.repo.MMSIExists(position.MMSI)
	if err != nil {
		log.Printf("❌ [Worker %d] Error verificando existencia de barco MMSI %d: %v", id, position.MMSI, err)
	}
	// Si el barco no existe, se almacena
	if exists {
		log.Printf("%s[Worker %d]%s 📍 Posición registrada para MMSI: %d", colorGreen, id, colorReset, position.MMSI)
		c.repo.SavePosition(&position)
	}
}
