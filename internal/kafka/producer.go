package kafka

import (
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
)

// Producer representa un productor de Kafka
type Producer struct {
	producer sarama.SyncProducer
}

// NewProducer crea un nuevo productor de Kafka
func NewProducer(brokers []string) (*Producer, error) {
	config := sarama.NewConfig()

	// Configuracion para robustez
	config.Producer.RequiredAcks = sarama.WaitForAll       // Espera a que todos los brokers confirmen la recepcion del mensaje
	config.Producer.Retry.Max = 5                          // Intenta reenviar el mensaje 5 veces
	config.Producer.Retry.Backoff = 100 * time.Millisecond // Espera 100 ms antes de reintentar
	config.Producer.Return.Successes = true                // Retorna los mensajes exitosos
	config.Producer.Idempotent = true                      // Habilita el modo idempotente para evitar duplicados
	config.Net.MaxOpenRequests = 1                         // Limita el numero de peticiones abiertas

	p, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("Error creando productor de Kafka: %w", err)
	}

	return &Producer{producer: p}, nil
}

// SendMessage envia un mensaje a un tópico de Kafka
func (p *Producer) SendMessage(topic, message string) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("Error enviando mensaje a %s: %w", topic, err)
	}
	log.Printf("✅ Mensaje enviado a topico %s [P: %d, O:%d]", topic, partition, offset)
	return nil
}

// Close cierra el productor de Kafka
func (p *Producer) Close() error {
	return p.producer.Close()
}
