package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config almacena toda la configuración de la aplicación cargada con Viper
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Kafka         KafkaConfig         `mapstructure:"kafka"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	Log           LogConfig           `mapstructure:"log"`
}

type ServerConfig struct {
	Host       string `mapstructure:"host"`
	Port       string `mapstructure:"port"`
	CorsOrigin string `mapstructure:"cors_origin"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

type KafkaConfig struct {
	Brokers      []string `mapstructure:"brokers"`
	StaticTopic  string   `mapstructure:"static_topic"`
	DynamicTopic string   `mapstructure:"dynamic_topic"`
	GroupID      string   `mapstructure:"group_id"`
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

type ObservabilityConfig struct {
	ServiceName  string `mapstructure:"service_name"`
	OTLPEndpoint string `mapstructure:"otlp_endpoint"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

// GetServerAddr devuelve la dirección de escucha formateada (host:puerto)
func (c *Config) GetServerAddr() string {
	port := c.Server.Port
	if port == "" {
		port = "8080"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

// Load lee el archivo config.yaml (si existe) y se superpone con las variables de entorno
func Load() (*Config, error) {
	v := viper.New()

	// Config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./apps/server")

	// Environment variable overrides
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Default values
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.cors_origin", "*")
	v.SetDefault("database.dsn", "host=localhost user=magi password=secret dbname=magi port=5432 sslmode=disable")
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.static_topic", "ais.static")
	v.SetDefault("kafka.dynamic_topic", "ais.dynamic")
	v.SetDefault("kafka.group_id", "magi-consumer-group")
	v.SetDefault("redis.url", "redis://127.0.0.1:6379")
	v.SetDefault("observability.service_name", "magi-service")
	v.SetDefault("observability.otlp_endpoint", "http://localhost:4318")
	v.SetDefault("log.level", "debug")

	_ = v.ReadInConfig()

	// Direct environment variable mapping for common twelve-factor patterns
	if p := os.Getenv("PORT"); p != "" {
		v.Set("server.port", p)
	}
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		v.Set("database.dsn", dsn)
	} else if dbUrl := os.Getenv("DATABASE_URL"); dbUrl != "" {
		v.Set("database.dsn", dbUrl)
	}
	if kb := os.Getenv("KAFKA_BROKERS"); kb != "" {
		v.Set("kafka.brokers", strings.Split(kb, ","))
	}
	if st := os.Getenv("KAFKA_STATIC_TOPIC"); st != "" {
		v.Set("kafka.static_topic", st)
	}
	if dt := os.Getenv("KAFKA_DYNAMIC_TOPIC"); dt != "" {
		v.Set("kafka.dynamic_topic", dt)
	}
	if kg := os.Getenv("KAFKA_GROUP_ID"); kg != "" {
		v.Set("kafka.group_id", kg)
	}
	if rUrl := os.Getenv("REDIS_URL"); rUrl != "" {
		v.Set("redis.url", rUrl)
	}
	if cors := os.Getenv("CORS_ORIGIN"); cors != "" {
		v.Set("server.cors_origin", cors)
	}
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		v.Set("log.level", lvl)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
