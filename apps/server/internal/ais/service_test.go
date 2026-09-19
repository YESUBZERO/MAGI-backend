package ais_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aires-magi/internal/ais"
	"aires-magi/internal/models"
)

type mockRepository struct {
	saveStaticFn  func(msg *models.StaticAIS) error
	saveDynamicFn func(msg *models.DynamicAIS) error
	getByIMOFn    func(imo int, limit, offset int) (*models.StaticAIS, error)
	getByMMSIFn   func(mmsi int, limit, offset int) (*models.StaticAIS, error)
}

func (m *mockRepository) SaveStatic(msg *models.StaticAIS) error {
	if m.saveStaticFn != nil {
		return m.saveStaticFn(msg)
	}
	return nil
}

func (m *mockRepository) SaveDynamic(msg *models.DynamicAIS) error {
	if m.saveDynamicFn != nil {
		return m.saveDynamicFn(msg)
	}
	return nil
}

func (m *mockRepository) GetByIMO(imo int, limit, offset int) (*models.StaticAIS, error) {
	if m.getByIMOFn != nil {
		return m.getByIMOFn(imo, limit, offset)
	}
	return nil, nil
}

func (m *mockRepository) GetByMMSI(mmsi int, limit, offset int) (*models.StaticAIS, error) {
	if m.getByMMSIFn != nil {
		return m.getByMMSIFn(mmsi, limit, offset)
	}
	return nil, nil
}

func TestProcessStaticMessage_Success(t *testing.T) {
	var savedMsg *models.StaticAIS
	mockRepo := &mockRepository{
		saveStaticFn: func(msg *models.StaticAIS) error {
			savedMsg = msg
			return nil
		},
	}
	svc := ais.NewService(mockRepo)

	msg := &models.StaticAIS{
		MMSI:     244670000,
		IMO:      9241061,
		Shipname: "SEASPAN EMPIRE",
		Callsign: "VRGO2",
		ShipType: "Cargo",
	}

	err := svc.ProcessStaticMessage(msg)
	require.NoError(t, err)
	require.NotNil(t, savedMsg)
	assert.Equal(t, "SEASPAN EMPIRE", savedMsg.Shipname)
	assert.Equal(t, 244670000, savedMsg.MMSI)
}

func TestProcessStaticMessage_Nil(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	err := svc.ProcessStaticMessage(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no puede ser nulo")
}

func TestProcessStaticMessage_InvalidMMSI(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	msg := &models.StaticAIS{MMSI: 0, IMO: 9241061}
	err := svc.ProcessStaticMessage(msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MMSI")
}

func TestProcessStaticMessage_InvalidIMO(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	msg := &models.StaticAIS{MMSI: 244670000, IMO: 12345} // < 7 dígitos
	err := svc.ProcessStaticMessage(msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IMO no es válido")
}

func TestProcessDynamicMessage_Success(t *testing.T) {
	var savedMsg *models.DynamicAIS
	mockRepo := &mockRepository{
		saveDynamicFn: func(msg *models.DynamicAIS) error {
			savedMsg = msg
			return nil
		},
	}
	svc := ais.NewService(mockRepo)

	now := time.Now()
	msg := &models.DynamicAIS{
		MMSI:      244670000,
		Timestamp: now,
		Latitude:  -33.0245,
		Longitude: -71.6212,
		Speed:     14.2,
		Course:    185.0,
		Heading:   184,
	}

	err := svc.ProcessDynamicMessage(msg)
	require.NoError(t, err)
	require.NotNil(t, savedMsg)
	assert.InDelta(t, -33.0245, savedMsg.Latitude, 0.0001)
	assert.InDelta(t, -71.6212, savedMsg.Longitude, 0.0001)
}

func TestProcessDynamicMessage_InvalidCoordinates(t *testing.T) {
	svc := ais.NewService(&mockRepository{})

	// Latitud fuera de rango (-90 a 90)
	msgLat := &models.DynamicAIS{MMSI: 244670000, Latitude: 95.0, Longitude: 10.0}
	errLat := svc.ProcessDynamicMessage(msgLat)
	require.Error(t, errLat)
	assert.Contains(t, errLat.Error(), "latitud")

	// Longitud fuera de rango (-180 a 180)
	msgLon := &models.DynamicAIS{MMSI: 244670000, Latitude: 10.0, Longitude: 195.0}
	errLon := svc.ProcessDynamicMessage(msgLon)
	require.Error(t, errLon)
	assert.Contains(t, errLon.Error(), "longitud")
}

func TestGetShipByIMO_Success(t *testing.T) {
	mockRepo := &mockRepository{
		getByIMOFn: func(imo int, limit, offset int) (*models.StaticAIS, error) {
			return &models.StaticAIS{
				IMO:      imo,
				MMSI:     244670000,
				Shipname: "SEASPAN EMPIRE",
			}, nil
		},
	}
	svc := ais.NewService(mockRepo)

	ship, err := svc.GetShipByIMO(9241061, 10, 0)
	require.NoError(t, err)
	require.NotNil(t, ship)
	assert.Equal(t, 9241061, ship.IMO)
	assert.Equal(t, "SEASPAN EMPIRE", ship.Shipname)
}

func TestGetShipByIMO_NotFound(t *testing.T) {
	mockRepo := &mockRepository{
		getByIMOFn: func(imo int, limit, offset int) (*models.StaticAIS, error) {
			return nil, models.ErrShipNotFound
		},
	}
	svc := ais.NewService(mockRepo)

	_, err := svc.GetShipByIMO(9241061, 10, 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrShipNotFound))
}
