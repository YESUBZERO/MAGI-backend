package config_test

import (
	"os"
	"testing"

	"github.com/YESUBZERO/consumer-service/internal/config"
)

func TestConfig_GetServerPort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Puerto por defecto si está vacío", "", ":8080"},
		{"Puerto sin dos puntos", "3000", ":3000"},
		{"Puerto con dos puntos", ":9090", ":9090"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{Port: tc.input},
			}
			if got := cfg.GetServerPort(); got != tc.expected {
				t.Errorf("se esperaba %s, se obtuvo: %s", tc.expected, got)
			}
		})
	}
}

func TestConfig_Load_ValidationFailure(t *testing.T) {
	// Limpiamos variables de entorno requeridas
	os.Unsetenv("KAFKA_BROKERS")
	os.Unsetenv("KAFKA_STATIC_TOPIC")
	os.Unsetenv("KAFKA_DYNAMIC_TOPIC")
	os.Unsetenv("KAFKA_GROUP_ID")
	os.Unsetenv("DATABASE_DSN")

	_, err := config.Load()
	if err == nil {
		t.Fatal("se esperaba error de validación cuando faltan variables de entorno obligatorias")
	}
}

func TestConfig_Load_Success(t *testing.T) {
	os.Setenv("KAFKA_BROKERS", "localhost:9092")
	os.Setenv("KAFKA_STATIC_TOPIC", "ais.static")
	os.Setenv("KAFKA_DYNAMIC_TOPIC", "ais.dynamic")
	os.Setenv("KAFKA_GROUP_ID", "magi-group")
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/magi?sslmode=disable")
	os.Setenv("PORT", "8888")
	defer func() {
		os.Unsetenv("KAFKA_BROKERS")
		os.Unsetenv("KAFKA_STATIC_TOPIC")
		os.Unsetenv("KAFKA_DYNAMIC_TOPIC")
		os.Unsetenv("KAFKA_GROUP_ID")
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("PORT")
	}()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("error inesperado cargando configuración: %v", err)
	}

	if len(cfg.GetKafkaBrokers()) != 1 || cfg.GetKafkaBrokers()[0] != "localhost:9092" {
		t.Errorf("brokers incorrectos: %v", cfg.GetKafkaBrokers())
	}
	static, dynamic := cfg.GetKafkaTopics()
	if static != "ais.static" || dynamic != "ais.dynamic" {
		t.Errorf("tópicos incorrectos: %s, %s", static, dynamic)
	}
	if cfg.GetKafkaGroupID() != "magi-group" {
		t.Errorf("groupID incorrecto: %s", cfg.GetKafkaGroupID())
	}
	if cfg.GetDSN() != "postgres://user:pass@localhost:5432/magi?sslmode=disable" {
		t.Errorf("DSN incorrecto: %s", cfg.GetDSN())
	}
	if cfg.GetServerPort() != ":8888" {
		t.Errorf("puerto incorrecto: %s", cfg.GetServerPort())
	}
}

func TestConfig_LoadAPI_Success(t *testing.T) {
	// Solo definimos DATABASE_DSN y PORT, sin variables de Kafka
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/magi?sslmode=disable")
	os.Setenv("PORT", "9000")
	defer func() {
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("PORT")
	}()

	cfg, err := config.LoadAPI()
	if err != nil {
		t.Fatalf("error inesperado cargando configuración de API: %v", err)
	}

	if cfg.GetDSN() != "postgres://user:pass@localhost:5432/magi?sslmode=disable" {
		t.Errorf("DSN incorrecto: %s", cfg.GetDSN())
	}
	if cfg.GetServerPort() != ":9000" {
		t.Errorf("puerto incorrecto: %s", cfg.GetServerPort())
	}
}

func TestConfig_LoadAPI_MissingDSN(t *testing.T) {
	os.Unsetenv("DATABASE_DSN")

	_, err := config.LoadAPI()
	if err == nil {
		t.Fatal("se esperaba error cuando DATABASE_DSN no está configurado")
	}
}
