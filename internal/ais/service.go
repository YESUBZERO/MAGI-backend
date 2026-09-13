package ais

import "errors"

// ===========================================================
// INTERFACES Y CONTRATOS DEL SERVICIO (CQRS)
// ===========================================================

// IngestionService define las operaciones de negocio para la ingesta y validación de telemetría AIS.
type IngestionService interface {
	ProcessStaticMessage(ais *StaticAIS) error
	ProcessDynamicMessage(ais *DynamicAIS) error
}

// QueryService define las operaciones de negocio para la consulta y lectura de buques y telemetría.
type QueryService interface {
	GetShipByIMO(imo int, limit, offset int) (*StaticAIS, error)
	GetShipByMMSI(mmsi int, limit, offset int) (*StaticAIS, error)
}

// Service compone IngestionService y QueryService para componentes que requieran ambas responsabilidades.
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

// NewIngestionService construye el servicio dedicado a la ingesta y validación de mensajes AIS.
func NewIngestionService(repo Repository) IngestionService {
	return &ingestionService{repo: repo}
}

// ProcessStaticMessage valida y persiste mensajes AIS estáticos (identidad y viaje).
func (s *ingestionService) ProcessStaticMessage(msg *StaticAIS) error {
	if msg == nil {
		return errors.New("el mensaje estático no puede ser nulo")
	}

	// 1. El MMSI es el identificador marítimo obligatorio
	if msg.MMSI <= 0 {
		return errors.New("el número MMSI es obligatorio")
	}

	// 2. El IMO debe ser un número válido de 7 dígitos
	if msg.IMO < 1000000 || msg.IMO > 9999999 {
		return errors.New("el número IMO no es válido")
	}

	// 3. Asegurar que el nombre del barco no guarde espacios vacíos
	if msg.Shipname == "" {
		msg.Shipname = "UNKNOWN"
	}

	// 4. Enviamos al repositorio tras las validaciones.
	return s.repo.SaveStatic(msg)
}

// ProcessDynamicMessage valida y persiste mensajes AIS dinámicos (posición, velocidad y rumbo).
func (s *ingestionService) ProcessDynamicMessage(msg *DynamicAIS) error {
	if msg == nil {
		return errors.New("el mensaje dinámico no puede ser nulo")
	}

	// 1. Validar el identificador MMSI del buque
	if msg.MMSI <= 0 {
		return errors.New("el código MMSI es obligatorio para registrar telemetría")
	}

	// 2. Validar el rango geográfico según norma ITU-R M.1371
	if msg.Latitude < -90.0 || msg.Latitude > 90.0 {
		return errors.New("la latitud debe estar comprendida entre -90 y 90 grados")
	}
	if msg.Longitude < -180.0 || msg.Longitude > 180.0 {
		return errors.New("la longitud debe estar comprendida entre -180 y 180 grados")
	}

	// 3. La velocidad de un barco en nudos no puede ser negativa
	if msg.Speed < 0 {
		return errors.New("la velocidad en nudos no puede ser negativa")
	}

	// 4. Enviamos al repositorio
	return s.repo.SaveDynamic(msg)
}

// ===========================================================
// IMPLEMENTACIÓN DE CONSULTAS (Read Path)
// ===========================================================

type queryService struct {
	repo Repository
}

// NewQueryService construye el servicio dedicado a la consulta y lectura paginada de buques.
func NewQueryService(repo Repository) QueryService {
	return &queryService{repo: repo}
}

// GetShipByIMO obtiene la información de un buque por su número IMO con telemetría paginada.
func (s *queryService) GetShipByIMO(imo int, limit, offset int) (*StaticAIS, error) {
	// 1. El número IMO proporcionado debe ser válido
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

// GetShipByMMSI obtiene la información de un buque por su código MMSI con telemetría paginada.
func (s *queryService) GetShipByMMSI(mmsi int, limit, offset int) (*StaticAIS, error) {
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
// COMPOSICIÓN (Para compatibilidad y casos unificados)
// ===========================================================

type compositeService struct {
	IngestionService
	QueryService
}

// NewService compone IngestionService y QueryService bajo la interfaz unificada Service.
func NewService(repo Repository) Service {
	return &compositeService{
		IngestionService: NewIngestionService(repo),
		QueryService:     NewQueryService(repo),
	}
}
