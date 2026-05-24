package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Configuración principal del servicio
type Config struct {
	Kafka KafkaConfig
	DB    DatabaseConfig
}

// Configuración de Kafka
type KafkaConfig struct {
	Brokers      []string `envconfig:"KAFKA_BROKERS" required:"true"`
	StaticTopic  string   `envconfig:"KAFKA_STATIC_TOPIC" required:"true"`
	DynamicTopic string   `envconfig:"KAFKA_DYNAMIC_TOPIC" required:"true"`
	GroupID      string   `envconfig:"KAFKA_GROUP_ID" required:"true"`
}

// DatabaseConfig representa la configuración de la base de datos
type DatabaseConfig struct {
	DSN string `envconfig:"DATABASE_DSN" required:"true"`
}

// GetKafkaBrokers devuelve los brokers de Kafka
func (c *Config) GetKafkaBrokers() []string {
	return c.Kafka.Brokers
}

// GetKafkaTopics devuelve los topics de Kafka
func (c *Config) GetKafkaTopics() (string, string) {
	return c.Kafka.StaticTopic, c.Kafka.DynamicTopic
}

// GetKafkaGroupID devuelve el Consumer Group ID de Kafka
func (c *Config) GetKafkaGroupID() string {
	return c.Kafka.GroupID
}

// GetDSN devuelve el DSN de la base de datos
func (c *Config) GetDSN() string {
	return c.DB.DSN
}

// Validate verifica que los campos esenciales no esten vacios
func (c *Config) validate() error {
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	if c.Kafka.StaticTopic == "" {
		return fmt.Errorf("KAFKA_STATIC_TOPIC is required")
	}
	if c.Kafka.DynamicTopic == "" {
		return fmt.Errorf("KAFKA_DYNAMIC_TOPIC is required")
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

	// 1. Intentar cargar el archivo .env si existe (Solo para desarrollo)
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error cargando archivo .env: %w", err)
	}

	cfg := &Config{}

	// 2. Mapear las variables de entorno a la estructura Config
	//    Usamos envconfig para facilitar el mapeo y validación de variables de entorno
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("error mapeando variables de entorno: %w", err)
	}

	// 3. Validar la configuración cargada
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("la validación de configuración falló: %w", err)
	}

	return cfg, nil
}
