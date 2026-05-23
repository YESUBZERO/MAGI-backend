package kafka

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/IBM/sarama"
	"github.com/YESUBZERO/consumer-service/internal/ais"
)

// WorkerPool define la cantidad de workers concurrentes
const (
	colorReset = "\033[0m"
	colorGreen = "\033[32m"
	colorBlue  = "\033[34m"
	WorkerPool = 5
)

// ===========================================================
// INTERFAZ DE CONFIGURACIÓN
// ===========================================================

// KafkaConfigReader define la interfaz para la configuración de Kafka
type KafkaConfigReader interface {
	GetKafkaBrokers() []string
	GetKafkaTopics() (string, string)
	GetKafkaGroupID() string
}

// ===========================================================
// CONSUMER (orquestador principal)
// ===========================================================

// Consumer orquesta el Consumer Group de Kafka y el worker pool
type Consumer struct {
	cfg        KafkaConfigReader
	aisService ais.Service
}

// NewConsumer crea un nuevo Consumer
func NewConsumer(cr KafkaConfigReader, as ais.Service) *Consumer {
	return &Consumer{
		cfg:        cr,
		aisService: as,
	}
}

// Start inicia el Consumer Group y el worker pool. Bloquea hasta que ctx se cancele.
func (c *Consumer) Start(ctx context.Context) {
	brokers := c.cfg.GetKafkaBrokers()
	staticTopic, dynamicTopic := c.cfg.GetKafkaTopics()
	groupID := c.cfg.GetKafkaGroupID()
	topics := []string{staticTopic, dynamicTopic}

	// Canal compartido entre el ConsumerGroupHandler y el worker pool
	messageChannel := make(chan *sarama.ConsumerMessage, 500)
	var wg sync.WaitGroup

	// 1. Iniciar el worker pool
	for i := range WorkerPool {
		wg.Add(1)
		go c.worker(i, &wg, messageChannel, staticTopic, dynamicTopic)
	}

	// 2. Configurar sarama para el Consumer Group
	saramaCfg := sarama.NewConfig()
	saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	// Commit automático de offsets al marcar los mensajes con session.MarkMessage()
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = true
	// Empezar desde el offset más antiguo disponible si no hay offset comprometido previo
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	// 3. Crear el Consumer Group
	group, err := sarama.NewConsumerGroup(brokers, groupID, saramaCfg)
	if err != nil {
		log.Fatalf("❌ Error creando Consumer Group de Kafka: %v", err)
	}
	defer group.Close()

	// 4. Handler que implementa sarama.ConsumerGroupHandler
	handler := &consumerGroupHandler{messageChannel: messageChannel}

	log.Printf("📡 [MAGI-CORE] Consumer Group '%s' escuchando tópicos: %v", groupID, topics)

	// 5. Loop de consume — se relanza tras cada rebalanceo del grupo
	go func() {
		for {
			// Consume() bloquea hasta que el contexto se cancele o haya un rebalanceo.
			// Tras un rebalanceo el loop vuelve a llamar Consume() automáticamente.
			if err := group.Consume(ctx, topics, handler); err != nil {
				log.Printf("⚠️ Error en Consumer Group: %v", err)
			}
			// Si el contexto fue cancelado, salimos del loop
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// 6. Esperar la señal de cierre
	<-ctx.Done()
	log.Println("🛑 Apagando Consumer Group de Kafka de forma segura...")
	close(messageChannel)
	wg.Wait()
	log.Println("✅ Todos los workers de Kafka han terminado.")
}

// ===========================================================
// CONSUMER GROUP HANDLER
// ===========================================================

// consumerGroupHandler implementa sarama.ConsumerGroupHandler.
// Su única responsabilidad es reenviar los mensajes recibidos al canal
// del worker pool y confirmar el offset una vez entregados.
type consumerGroupHandler struct {
	messageChannel chan<- *sarama.ConsumerMessage
}

// Setup se ejecuta al inicio de cada sesión de Consumer Group (tras un rebalanceo).
func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	log.Printf("🔄 [Consumer Group] Sesión iniciada — miembro: %s", session.MemberID())
	return nil
}

// Cleanup se ejecuta al final de cada sesión de Consumer Group.
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Println("🔄 [Consumer Group] Sesión finalizada.")
	return nil
}

// ConsumeClaim lee los mensajes de una partición asignada y los envía al worker pool.
// Llama a session.MarkMessage() para comprometer el offset tras entregar cada mensaje.
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				// El canal se cerró (rebalanceo o shutdown)
				return nil
			}
			h.messageChannel <- msg
			// Marcar el offset como procesado para que Kafka no lo reenvíe
			session.MarkMessage(msg, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// ===========================================================
// WORKER POOL
// ===========================================================

// worker procesa mensajes del canal y los delega al servicio AIS correspondiente
func (c *Consumer) worker(id int, wg *sync.WaitGroup, ch <-chan *sarama.ConsumerMessage, staticTopic, dynamicTopic string) {
	defer wg.Done()

	for msg := range ch {
		switch msg.Topic {
		case staticTopic:
			var staticMsg ais.StaticAIS
			if err := json.Unmarshal(msg.Value, &staticMsg); err != nil {
				log.Printf("❌ [Worker %d] Error parseando estático: %v", id, err)
				continue
			}
			c.processStatic(id, &staticMsg)

		case dynamicTopic:
			var dynamicMsg ais.DynamicAIS
			if err := json.Unmarshal(msg.Value, &dynamicMsg); err != nil {
				log.Printf("❌ [Worker %d] Error parseando dinámico: %v", id, err)
				continue
			}
			c.processDynamic(id, &dynamicMsg)
		}
	}
}

// processStatic procesa mensajes AIS estáticos
func (c *Consumer) processStatic(workerID int, msg *ais.StaticAIS) {
	log.Printf("%s[Worker %d]%s 🚢 Procesando Estático (MMSI: %d)", colorBlue, workerID, colorReset, msg.MMSI)

	if err := c.aisService.ProcessStaticMessage(msg); err != nil {
		log.Printf("❌ [Worker %d] Error procesando estático: %v", workerID, err)
	}
}

// processDynamic procesa mensajes AIS dinámicos
func (c *Consumer) processDynamic(workerID int, msg *ais.DynamicAIS) {
	log.Printf("%s[Worker %d]%s 🚢 Procesando Dinámico (MMSI: %d)", colorGreen, workerID, colorReset, msg.MMSI)

	if err := c.aisService.ProcessDynamicMessage(msg); err != nil {
		log.Printf("❌ [Worker %d] Error procesando dinámico: %v", workerID, err)
	}
}
