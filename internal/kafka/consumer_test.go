package kafka

import (
	"testing"

	"github.com/YESUBZERO/consumer-service/internal/models"
)

// Mock que cumple con la interfaz de ShipStorer
type MockShipRepo struct {
	existCalled bool
	saveCalled  bool
	existReturn bool
}

func (m *MockShipRepo) ShipExists(imo int) (bool, error) {
	m.existCalled = true
	return m.existReturn, nil
}

func (m *MockShipRepo) SaveShip(ship *models.Ship) error {
	m.saveCalled = true
	return nil
}

// Mock para la configuracion de Kafka
type MockConfig struct{}

func (m *MockConfig) GetKafkaBrokers() []string {
	return []string{"localhost:9092"}
}

func (m *MockConfig) GetKafkaTopics() (string, string, string) {
	return "static-message", "scrape-message", "enriched-message"
}

// Test
func TestWorker_SaveShip(t *testing.T) {
	// preparar
	mockRepo := &MockShipRepo{existReturn: false}
	mockConfig := &MockConfig{}

	// No necesitamos un productor real
	consumer := NewConsumer(mockConfig, nil, mockRepo)

	// Simulamos un mensaje
	ship := models.Ship{IMO: 123456, Shipname: "Test Ship"}

	// Probar worker
	exist, _ := consumer.repo.ShipExists(ship.IMO)
	if !exist {
		consumer.repo.SaveShip(&ship)
	}

	// Verificar (Assert)
	if !mockRepo.existCalled {
		t.Error("ShipExists no fue llamado")
	}
	if !mockRepo.saveCalled {
		t.Error("SaveShip no fue llamado")
	}
}
