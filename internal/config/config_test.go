package config

import (
	"os"
	"testing"
)

func TestConfig_Load(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("KAFKA_BROKERS", "localhost:9092")
	os.Setenv("KAFKA_STATIC_TOPIC", "test-topic")
	os.Setenv("KAFKA_GROUP_ID", "test-group")
	os.Setenv("DATABASE_DSN", "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(cfg.Kafka.Brokers) == 0 || cfg.Kafka.Brokers[0] != "localhost:9092" {
		t.Errorf("Expected broker localhost:9092, got %v", cfg.Kafka.Brokers)
	}

	// Validamos campos clave para asegurar que envconfig está funcionando
	if cfg.Kafka.StaticTopic != "test-topic" {
		t.Errorf("Esperado topic 'test-topic', obtenido '%s'", cfg.Kafka.StaticTopic)
	}

	if cfg.Kafka.GroupID != "test-group" {
		t.Errorf("Esperado group 'test-group', obtenido '%s'", cfg.Kafka.GroupID)
	}

	if cfg.DB.DSN == "" {
		t.Error("Expected DSN to be set")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "Valid config",
			cfg: Config{
				Kafka: KafkaConfig{
					Brokers:     []string{"broker1"},
					StaticTopic: "topic1",
					GroupID:     "group1",
				},
				DB: DatabaseConfig{
					DSN: "dsn1",
				},
			},
			wantErr: false,
		},
		{
			name: "Missing Kafka brokers",
			cfg: Config{
				Kafka: KafkaConfig{
					StaticTopic: "topic1",
					GroupID:     "group1",
				},
				DB: DatabaseConfig{
					DSN: "dsn1",
				},
			},
			wantErr: true,
		},
		{
			name: "Missing DB DSN",
			cfg: Config{
				Kafka: KafkaConfig{
					Brokers:     []string{"broker1"},
					StaticTopic: "topic1",
					GroupID:     "group1",
				},
				DB: DatabaseConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
