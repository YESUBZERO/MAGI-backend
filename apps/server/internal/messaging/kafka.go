package messaging

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"aires-magi/internal/ais"
	"aires-magi/internal/config"
	"aires-magi/internal/models"
)

const DefaultWorkerPool = 5

// Producer encapsula la publicación de mensajes hacia Kafka
type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *Producer) Publish(ctx context.Context, key string, payload interface{}) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: bytes,
		Time:  time.Now(),
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

// Consumer gestiona el Consumer Group de Kafka con worker pool concurrente y semántica at-least-once
type Consumer struct {
	cfg        *config.KafkaConfig
	aisService ais.IngestionService
	logger     zerolog.Logger
	workerPool int
}

// NewConsumer crea un nuevo Consumer de Kafka
func NewConsumer(cfg *config.KafkaConfig, svc ais.IngestionService, zlog zerolog.Logger) *Consumer {
	return &Consumer{
		cfg:        cfg,
		aisService: svc,
		logger:     zlog.With().Str("component", "kafka-consumer").Logger(),
		workerPool: DefaultWorkerPool,
	}
}

type kafkaJob struct {
	msg kafka.Message
}

// Start inicia la lectura de mensajes del Consumer Group y bloquea hasta que ctx se cancele
func (c *Consumer) Start(ctx context.Context) {
	topics := []string{c.cfg.StaticTopic, c.cfg.DynamicTopic}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.cfg.Brokers,
		GroupID:        c.cfg.GroupID,
		GroupTopics:    topics,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: 0,    // Commit manual explícito tras procesar
		StartOffset:    kafka.FirstOffset,
	})
	defer reader.Close()

	c.logger.Info().
		Strs("brokers", c.cfg.Brokers).
		Str("groupID", c.cfg.GroupID).
		Strs("topics", topics).
		Int("workers", c.workerPool).
		Msg("Consumer Group de Kafka inicializado y listo")

	jobChannel := make(chan *kafkaJob, 500)
	var wg sync.WaitGroup

	// Iniciar worker pool concurrente
	for i := 0; i < c.workerPool; i++ {
		wg.Add(1)
		go c.worker(i, &wg, reader, jobChannel)
	}

	// Goroutine para lectura continua de Kafka
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error().Err(err).Msg("Error al obtener mensaje de Kafka")
				time.Sleep(500 * time.Millisecond)
				continue
			}

			select {
			case jobChannel <- &kafkaJob{msg: msg}:
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	c.logger.Info().Msg("Señal de apagado recibida en Kafka consumer. Drenando workers...")

	<-readerDone
	close(jobChannel)
	wg.Wait()
	c.logger.Info().Msg("Todos los workers de Kafka han finalizado limpiamente")
}

func (c *Consumer) worker(id int, wg *sync.WaitGroup, reader *kafka.Reader, ch <-chan *kafkaJob) {
	defer wg.Done()

	for job := range ch {
		msg := job.msg
		var shouldCommit bool

		switch msg.Topic {
		case c.cfg.StaticTopic:
			var staticMsg models.StaticAIS
			if err := json.Unmarshal(msg.Value, &staticMsg); err != nil {
				c.logger.Warn().Err(err).Int("worker", id).Msg("Mensaje estático corrupto. Descartando y confirmando offset.")
				shouldCommit = true
			} else {
				if err := c.aisService.ProcessStaticMessage(&staticMsg); err != nil {
					c.logger.Error().Err(err).Int("worker", id).Int("mmsi", staticMsg.MMSI).Msg("Error persistiendo buque estático")
				} else {
					c.logger.Debug().Int("worker", id).Int("mmsi", staticMsg.MMSI).Msg("Buque estático procesado con éxito")
					shouldCommit = true
				}
			}

		case c.cfg.DynamicTopic:
			var dynamicMsg models.DynamicAIS
			if err := json.Unmarshal(msg.Value, &dynamicMsg); err != nil {
				c.logger.Warn().Err(err).Int("worker", id).Msg("Mensaje dinámico corrupto. Descartando y confirmando offset.")
				shouldCommit = true
			} else {
				if err := c.aisService.ProcessDynamicMessage(&dynamicMsg); err != nil {
					c.logger.Error().Err(err).Int("worker", id).Int("mmsi", dynamicMsg.MMSI).Msg("Error persistiendo telemetría dinámica")
				} else {
					c.logger.Debug().Int("worker", id).Int("mmsi", dynamicMsg.MMSI).Msg("Telemetría dinámica procesada con éxito")
					shouldCommit = true
				}
			}
		}

		// Confirmamos el offset solo tras persistencia exitosa o descarte de mensaje corrupto (at-least-once)
		if shouldCommit {
			commitCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if err := reader.CommitMessages(commitCtx, msg); err != nil {
				c.logger.Error().Err(err).Int("worker", id).Msg("Error confirmando offset de Kafka")
			}
			cancel()
		}
	}
}
