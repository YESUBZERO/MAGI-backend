package ais_test

import (
	"errors"
	"testing"
	"time"

	"github.com/YESUBZERO/consumer-service/internal/ais"
)

// mockRepository simula la persistencia para pruebas unitarias
type mockRepository struct {
	saveStaticFn  func(msg *ais.StaticAIS) error
	saveDynamicFn func(msg *ais.DynamicAIS) error
	getByIMOFn    func(imo int, limit, offset int) (*ais.StaticAIS, error)
	getByMMSIFn   func(mmsi int, limit, offset int) (*ais.StaticAIS, error)
}

func (m *mockRepository) SaveStatic(msg *ais.StaticAIS) error {
	if m.saveStaticFn != nil {
		return m.saveStaticFn(msg)
	}
	return nil
}

func (m *mockRepository) SaveDynamic(msg *ais.DynamicAIS) error {
	if m.saveDynamicFn != nil {
		return m.saveDynamicFn(msg)
	}
	return nil
}

func (m *mockRepository) GetByIMO(imo int, limit, offset int) (*ais.StaticAIS, error) {
	if m.getByIMOFn != nil {
		return m.getByIMOFn(imo, limit, offset)
	}
	return nil, nil
}

func (m *mockRepository) GetByMMSI(mmsi int, limit, offset int) (*ais.StaticAIS, error) {
	if m.getByMMSIFn != nil {
		return m.getByMMSIFn(mmsi, limit, offset)
	}
	return nil, nil
}

// -------------------------------------------------------------
// Pruebas para ProcessStaticMessage
// -------------------------------------------------------------

func TestProcessStaticMessage_Success(t *testing.T) {
	var savedMsg *ais.StaticAIS
	mockRepo := &mockRepository{
		saveStaticFn: func(msg *ais.StaticAIS) error {
			savedMsg = msg
			return nil
		},
	}
	svc := ais.NewService(mockRepo)

	msg := &ais.StaticAIS{
		MMSI:     244670000,
		IMO:      9241061,
		Shipname: "SEASPAN EMPIRE",
		Callsign: "VRGO2",
		ShipType: "Cargo",
	}

	err := svc.ProcessStaticMessage(msg)
	if err != nil {
		t.Fatalf("se esperaba nil, se obtuvo error: %v", err)
	}
	if savedMsg == nil || savedMsg.Shipname != "SEASPAN EMPIRE" {
		t.Errorf("el mensaje guardado no coincide: %+v", savedMsg)
	}
}

func TestProcessStaticMessage_NilMsg(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	err := svc.ProcessStaticMessage(nil)
	if err == nil {
		t.Fatal("se esperaba error con mensaje nulo, se obtuvo nil")
	}
}

func TestProcessStaticMessage_InvalidMMSI(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	msg := &ais.StaticAIS{
		MMSI: 0,
		IMO:  9241061,
	}
	if err := svc.ProcessStaticMessage(msg); err == nil {
		t.Fatal("se esperaba error con MMSI <= 0, se obtuvo nil")
	}
}

func TestProcessStaticMessage_InvalidIMO(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	tests := []struct {
		name string
		imo  int
	}{
		{"IMO muy corto", 999999},
		{"IMO muy largo", 10000000},
		{"IMO negativo", -1234567},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &ais.StaticAIS{
				MMSI: 244670000,
				IMO:  tc.imo,
			}
			if err := svc.ProcessStaticMessage(msg); err == nil {
				t.Fatalf("se esperaba error para IMO %d", tc.imo)
			}
		})
	}
}

func TestProcessStaticMessage_EmptyShipnameDefaultToUNKNOWN(t *testing.T) {
	var savedMsg *ais.StaticAIS
	mockRepo := &mockRepository{
		saveStaticFn: func(msg *ais.StaticAIS) error {
			savedMsg = msg
			return nil
		},
	}
	svc := ais.NewService(mockRepo)

	msg := &ais.StaticAIS{
		MMSI:     244670000,
		IMO:      9241061,
		Shipname: "",
	}

	if err := svc.ProcessStaticMessage(msg); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if savedMsg.Shipname != "UNKNOWN" {
		t.Errorf("se esperaba 'UNKNOWN', se obtuvo: %s", savedMsg.Shipname)
	}
}

// -------------------------------------------------------------
// Pruebas para ProcessDynamicMessage
// -------------------------------------------------------------

func TestProcessDynamicMessage_Success(t *testing.T) {
	var savedMsg *ais.DynamicAIS
	mockRepo := &mockRepository{
		saveDynamicFn: func(msg *ais.DynamicAIS) error {
			savedMsg = msg
			return nil
		},
	}
	svc := ais.NewService(mockRepo)

	msg := &ais.DynamicAIS{
		MMSI:      244670000,
		Timestamp: time.Now(),
		Latitude:  12.3456,
		Longitude: -76.5432,
		Speed:     14.5,
		Course:    180.0,
	}

	if err := svc.ProcessDynamicMessage(msg); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if savedMsg == nil || savedMsg.MMSI != 244670000 {
		t.Errorf("el mensaje no se guardó correctamente: %+v", savedMsg)
	}
}

func TestProcessDynamicMessage_NilMsg(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	if err := svc.ProcessDynamicMessage(nil); err == nil {
		t.Fatal("se esperaba error al pasar mensaje dinámico nil")
	}
}

func TestProcessDynamicMessage_InvalidMMSI(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	msg := &ais.DynamicAIS{MMSI: 0}
	if err := svc.ProcessDynamicMessage(msg); err == nil {
		t.Fatal("se esperaba error con MMSI <= 0")
	}
}

func TestProcessDynamicMessage_GeographicBoundaries(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	tests := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{"Latitud mayor a 90", 90.1, 0.0},
		{"Latitud menor a -90", -90.1, 0.0},
		{"Longitud mayor a 180", 0.0, 180.1},
		{"Longitud menor a -180", 0.0, -180.1},
		{"Latitud AIS no disponible (91)", 91.0, 0.0},
		{"Longitud AIS no disponible (181)", 0.0, 181.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &ais.DynamicAIS{
				MMSI:      244670000,
				Latitude:  tc.lat,
				Longitude: tc.lon,
				Speed:     10.0,
			}
			if err := svc.ProcessDynamicMessage(msg); err == nil {
				t.Errorf("se esperaba error de coordenadas para (%v, %v)", tc.lat, tc.lon)
			}
		})
	}
}

func TestProcessDynamicMessage_NegativeSpeed(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	msg := &ais.DynamicAIS{
		MMSI:      244670000,
		Latitude:  10.0,
		Longitude: 10.0,
		Speed:     -1.0,
	}
	if err := svc.ProcessDynamicMessage(msg); err == nil {
		t.Fatal("se esperaba error para velocidad negativa")
	}
}

// -------------------------------------------------------------
// Pruebas para GetShipByIMO
// -------------------------------------------------------------

func TestGetShipByIMO_Success(t *testing.T) {
	expected := &ais.StaticAIS{
		IMO:      9241061,
		Shipname: "SEASPAN EMPIRE",
	}
	mockRepo := &mockRepository{
		getByIMOFn: func(imo int, limit, offset int) (*ais.StaticAIS, error) {
			if imo == 9241061 {
				return expected, nil
			}
			return nil, errors.New("not found")
		},
	}
	svc := ais.NewService(mockRepo)

	ship, err := svc.GetShipByIMO(9241061, 50, 0)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if ship.Shipname != "SEASPAN EMPIRE" {
		t.Errorf("se esperaba 'SEASPAN EMPIRE', se obtuvo: %s", ship.Shipname)
	}
}

func TestGetShipByIMO_InvalidIMO(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	if _, err := svc.GetShipByIMO(12345, 50, 0); err == nil {
		t.Fatal("se esperaba error para IMO inválido")
	}
}

func TestGetShipByMMSI_Success(t *testing.T) {
	expected := &ais.StaticAIS{
		MMSI:     244670000,
		Shipname: "SEASPAN EMPIRE",
	}
	mockRepo := &mockRepository{
		getByMMSIFn: func(mmsi int, limit, offset int) (*ais.StaticAIS, error) {
			if mmsi == 244670000 {
				return expected, nil
			}
			return nil, errors.New("not found")
		},
	}
	svc := ais.NewService(mockRepo)

	ship, err := svc.GetShipByMMSI(244670000, 50, 0)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if ship.Shipname != "SEASPAN EMPIRE" {
		t.Errorf("se esperaba 'SEASPAN EMPIRE', se obtuvo: %s", ship.Shipname)
	}
}

func TestGetShipByMMSI_InvalidMMSI(t *testing.T) {
	svc := ais.NewService(&mockRepository{})
	if _, err := svc.GetShipByMMSI(0, 50, 0); err == nil {
		t.Fatal("se esperaba error para MMSI <= 0")
	}
}

func TestNewIngestionService_ProcessMessage(t *testing.T) {
	var saved bool
	repo := &mockRepository{
		saveStaticFn: func(msg *ais.StaticAIS) error {
			saved = true
			return nil
		},
	}
	ingestionSvc := ais.NewIngestionService(repo)
	err := ingestionSvc.ProcessStaticMessage(&ais.StaticAIS{
		MMSI: 244670000,
		IMO:  9241061,
	})
	if err != nil {
		t.Fatalf("error inesperado en IngestionService: %v", err)
	}
	if !saved {
		t.Fatal("se esperaba que el mensaje fuera guardado")
	}
}

func TestNewQueryService_GetShip(t *testing.T) {
	repo := &mockRepository{
		getByIMOFn: func(imo int, limit, offset int) (*ais.StaticAIS, error) {
			return &ais.StaticAIS{IMO: imo, Shipname: "SEASPAN EMPIRE"}, nil
		},
	}
	querySvc := ais.NewQueryService(repo)
	ship, err := querySvc.GetShipByIMO(9241061, 50, 0)
	if err != nil {
		t.Fatalf("error inesperado en QueryService: %v", err)
	}
	if ship.Shipname != "SEASPAN EMPIRE" {
		t.Errorf("se esperaba 'SEASPAN EMPIRE', se obtuvo: %s", ship.Shipname)
	}
}

func TestIngestionService_ProcessDynamicMessage_Valid(t *testing.T) {
	var savedDynamic *ais.DynamicAIS
	repo := &mockRepository{
		saveDynamicFn: func(msg *ais.DynamicAIS) error {
			savedDynamic = msg
			return nil
		},
	}
	ingestionSvc := ais.NewIngestionService(repo)
	msg := &ais.DynamicAIS{
		MMSI:      244670000,
		Latitude:  -12.0464,
		Longitude: -77.0428,
		Speed:     14.5,
		Timestamp: time.Now(),
	}
	err := ingestionSvc.ProcessDynamicMessage(msg)
	if err != nil {
		t.Fatalf("error inesperado en IngestionService procesando dinámico: %v", err)
	}
	if savedDynamic == nil || savedDynamic.Speed != 14.5 {
		t.Fatalf("telemetría guardada no coincide: %+v", savedDynamic)
	}
}

func TestIngestionService_ProcessDynamicMessage_InvalidCoordinates(t *testing.T) {
	ingestionSvc := ais.NewIngestionService(&mockRepository{})
	msg := &ais.DynamicAIS{
		MMSI:      244670000,
		Latitude:  91.0, // ITU-R M.1371 Not Available
		Longitude: -77.0428,
		Speed:     10.0,
	}
	err := ingestionSvc.ProcessDynamicMessage(msg)
	if err == nil {
		t.Fatal("se esperaba error de latitud fuera de rango")
	}
}

func TestQueryService_GetShipByMMSI_Success(t *testing.T) {
	repo := &mockRepository{
		getByMMSIFn: func(mmsi int, limit, offset int) (*ais.StaticAIS, error) {
			return &ais.StaticAIS{MMSI: mmsi, Shipname: "OCEAN VOYAGER"}, nil
		},
	}
	querySvc := ais.NewQueryService(repo)
	ship, err := querySvc.GetShipByMMSI(244670000, 50, 0)
	if err != nil {
		t.Fatalf("error inesperado en QueryService: %v", err)
	}
	if ship.Shipname != "OCEAN VOYAGER" {
		t.Errorf("se esperaba 'OCEAN VOYAGER', se obtuvo: %s", ship.Shipname)
	}
}

func TestQueryService_PaginationNormalization(t *testing.T) {
	var capturedLimit, capturedOffset int
	repo := &mockRepository{
		getByIMOFn: func(imo int, limit, offset int) (*ais.StaticAIS, error) {
			capturedLimit = limit
			capturedOffset = offset
			return &ais.StaticAIS{IMO: imo}, nil
		},
	}
	querySvc := ais.NewQueryService(repo)

	// Caso 1: limit <= 0 -> normalizado a 50, offset < 0 -> normalizado a 0
	_, _ = querySvc.GetShipByIMO(9241061, -10, -5)
	if capturedLimit != 50 || capturedOffset != 0 {
		t.Errorf("esperado limit=50, offset=0; obtenido limit=%d, offset=%d", capturedLimit, capturedOffset)
	}

	// Caso 2: limit > 500 -> normalizado a 50
	_, _ = querySvc.GetShipByIMO(9241061, 9999, 100)
	if capturedLimit != 50 || capturedOffset != 100 {
		t.Errorf("esperado limit=50, offset=100; obtenido limit=%d, offset=%d", capturedLimit, capturedOffset)
	}
}
