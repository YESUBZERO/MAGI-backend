package ais

import (
	"errors"

	"aires-magi/internal/models"
)

// ===========================================================
// CONTRATOS CQRS
// ===========================================================

// IngestionService define las operaciones de negocio para la ingesta y validación de telemetría (Write Path)
type IngestionService interface {
	ProcessStaticMessage(ais *models.StaticAIS) error
	ProcessDynamicMessage(ais *models.DynamicAIS) error
}

// QueryService define las operaciones de negocio para la lectura de buques y telemetría (Read Path)
type QueryService interface {
	GetShipByIMO(imo int, limit, offset int) (*models.StaticAIS, error)
	GetShipByMMSI(mmsi int, limit, offset int) (*models.StaticAIS, error)
}

// Service compone IngestionService y QueryService para componentes que requieran ambas responsabilidades
type Service interface {
	IngestionService
	QueryService
}

// ===========================================================
// IMPLEMENTACIÓN DE INGESTIÓN (Write Path)
// ===========================================================

type ingestionService struct {
	repo Repository
}

// NewIngestionService construye el servicio dedicado a la ingesta y validación de mensajes AIS
func NewIngestionService(repo Repository) IngestionService {
	return &ingestionService{repo: repo}
}

// ProcessStaticMessage valida y persiste mensajes AIS estáticos (identidad y viaje)
func (s *ingestionService) ProcessStaticMessage(msg *models.StaticAIS) error {
	if msg == nil {
		return errors.New("el mensaje estático no puede ser nulo")
	}

	if msg.MMSI <= 0 {
		return errors.New("el número MMSI es obligatorio")
	}

	if msg.IMO < 1000000 || msg.IMO > 9999999 {
		return errors.New("el número IMO no es válido")
	}

	if msg.Shipname == "" {
		msg.Shipname = "UNKNOWN"
	}

	return s.repo.SaveStatic(msg)
}

// ProcessDynamicMessage valida y persiste mensajes AIS dinámicos según estándar marítimo ITU-R M.1371
func (s *ingestionService) ProcessDynamicMessage(msg *models.DynamicAIS) error {
	if msg == nil {
		return errors.New("el mensaje dinámico no puede ser nulo")
	}

	if msg.MMSI <= 0 {
		return errors.New("el código MMSI es obligatorio para registrar telemetría")
	}

	if msg.Latitude < -90.0 || msg.Latitude > 90.0 {
		return errors.New("la latitud debe estar comprendida entre -90 y 90 grados")
	}
	if msg.Longitude < -180.0 || msg.Longitude > 180.0 {
		return errors.New("la longitud debe estar comprendida entre -180 y 180 grados")
	}

	if msg.Speed < 0 {
		return errors.New("la velocidad en nudos no puede ser negativa")
	}

	return s.repo.SaveDynamic(msg)
}

// ===========================================================
// IMPLEMENTACIÓN DE CONSULTAS (Read Path)
// ===========================================================

type queryService struct {
	repo Repository
}

// NewQueryService construye el servicio dedicado a la consulta y lectura paginada de buques
func NewQueryService(repo Repository) QueryService {
	return &queryService{repo: repo}
}

// GetShipByIMO obtiene la información de un buque por su número IMO con telemetría paginada
func (s *queryService) GetShipByIMO(imo int, limit, offset int) (*models.StaticAIS, error) {
	if imo < 1000000 || imo > 9999999 {
		return nil, errors.New("el número IMO no es válido")
	}

	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetByIMO(imo, limit, offset)
}

// GetShipByMMSI obtiene la información de un buque por su código MMSI con telemetría paginada
func (s *queryService) GetShipByMMSI(mmsi int, limit, offset int) (*models.StaticAIS, error) {
	if mmsi <= 0 {
		return nil, errors.New("el código MMSI no es válido")
	}

	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetByMMSI(mmsi, limit, offset)
}

// ===========================================================
// COMPOSICIÓN (Servicio Unificado)
// ===========================================================

type compositeService struct {
	IngestionService
	QueryService
}

// NewService compone IngestionService y QueryService bajo la interfaz unificada Service
func NewService(repo Repository) Service {
	return &compositeService{
		IngestionService: NewIngestionService(repo),
		QueryService:     NewQueryService(repo),
	}
}
