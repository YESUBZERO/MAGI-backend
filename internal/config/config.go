package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Configuración principal del servicio
type Config struct {
	Kafka KafkaConfig
	DB    DatabaseConfig
}

// Configuración de Kafka
type KafkaConfig struct {
	Brokers       []string `envconfig:"KAFKA_BROKERS" required:"true"`
	StaticTopic   string   `envconfig:"KAFKA_STATIC_TOPIC" required:"true"`
	DynamicTopic  string   `envconfig:"KAFKA_DYNAMIC_TOPIC" required:"true"`
	ScrapeTopic   string   `envconfig:"KAFKA_SCRAPE_TOPIC" default:""`
	EnrichedTopic string   `envconfig:"KAFKA_ENRICHED_TOPIC" default:""`
	GroupID       string   `envconfig:"KAFKA_GROUP_ID" required:"true"`
}

type DatabaseConfig struct {
	DSN string `envconfig:"DATABASE_DSN" required:"true"`
}

// GetKafkaBrokers devuelve los brokers de Kafka
func (c *Config) GetKafkaBrokers() []string {
	return c.Kafka.Brokers
}

// GetKafkaTopics devuelve los topics de Kafka
func (c *Config) GetKafkaTopics() (string, string, string, string) {
	return c.Kafka.StaticTopic, c.Kafka.DynamicTopic, c.Kafka.ScrapeTopic, c.Kafka.EnrichedTopic
}

// GetDSN devuelve el DSN de la base de datos
func (c *Config) GetDSN() string {
	return c.DB.DSN
}

// Validate verifica que los campos esenciales no esten vacios
func (c *Config) Validate() error {
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	if c.Kafka.StaticTopic == "" {
		return fmt.Errorf("KAFKA_STATIC_TOPIC is required")
	}
	if c.Kafka.GroupID == "" {
		return fmt.Errorf("KAFKA_GROUP_ID is required")
	}
	if c.DB.DSN == "" {
		return fmt.Errorf("DATABASE_DSN is required")
	}
	return nil
}

// LoadConfig es un helper para mantener compatibilidad o carga simple
func Load() (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("error processing env vars: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}
