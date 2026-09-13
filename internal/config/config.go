package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Configuración principal del servicio
type Config struct {
	Kafka  KafkaConfig
	DB     DatabaseConfig
	Server ServerConfig
}

// ServerConfig representa la configuración del servidor HTTP
type ServerConfig struct {
	Port string `envconfig:"PORT" default:"8080"`
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

// GetServerPort devuelve el puerto configurado asegurando el formato ":puerto"
func (c *Config) GetServerPort() string {
	if c.Server.Port == "" {
		return ":8080"
	}
	if c.Server.Port[0] == ':' {
		return c.Server.Port
	}
	return ":" + c.Server.Port
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

// APIConfig representa la configuración exclusiva requerida por el microservicio HTTP de consultas
type APIConfig struct {
	DB     DatabaseConfig
	Server ServerConfig
}

// GetDSN devuelve el DSN de la base de datos para la API
func (c *APIConfig) GetDSN() string {
	return c.DB.DSN
}

// GetServerPort devuelve el puerto configurado asegurando el formato ":puerto"
func (c *APIConfig) GetServerPort() string {
	if c.Server.Port == "" {
		return ":8080"
	}
	if c.Server.Port[0] == ':' {
		return c.Server.Port
	}
	return ":" + c.Server.Port
}

func loadDotEnv() {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("ℹ️ Nota sobre archivo .env: %v", err)
		}
	}
}

// LoadAPI carga y valida exclusivamente las variables necesarias para el servidor HTTP (DATABASE_DSN y PORT)
func LoadAPI() (*APIConfig, error) {
	loadDotEnv()

	cfg := &APIConfig{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("error mapeando variables de entorno de la API: %w", err)
	}

	if cfg.DB.DSN == "" {
		return nil, fmt.Errorf("DATABASE_DSN is required")
	}

	return cfg, nil
}

// LoadWorker carga la configuración requerida para el ingestor de Kafka (brokers, tópicos y DSN)
func LoadWorker() (*Config, error) {
	return Load()
}

// Load es el cargador general para el worker y compatibilidad
func Load() (*Config, error) {
	loadDotEnv()

	cfg := &Config{}

	// Mapear las variables de entorno a la estructura Config
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("error mapeando variables de entorno: %w", err)
	}

	// Validar la configuración cargada
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("la validación de configuración falló: %w", err)
	}

	return cfg, nil
}
