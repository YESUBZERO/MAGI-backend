package ais

import (
	"context"
	"errors"
	"strings"

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

// ===========================================================
// BÚSQUEDA FACETADA Y TELEMETRÍA (Search Path)
// ===========================================================

// ShipSearchService define las operaciones de negocio para búsqueda facetada y telemetría acotada
type ShipSearchService interface {
	SearchShips(ctx context.Context, filter models.ShipSearchFilter) (*models.ShipSearchResponse, error)
	GetShipTelemetry(ctx context.Context, filter models.TelemetryQueryFilter) (*models.TelemetryQueryResponse, error)
}

type shipSearchService struct {
	repo SearchRepository
}

// NewShipSearchService construye el servicio para búsqueda facetada y consulta de telemetría acotada
func NewShipSearchService(repo SearchRepository) ShipSearchService {
	return &shipSearchService{repo: repo}
}

func (s *shipSearchService) SearchShips(ctx context.Context, filter models.ShipSearchFilter) (*models.ShipSearchResponse, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 500 {
		filter.Limit = 500
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.SearchShips(ctx, filter)
}

func (s *shipSearchService) GetShipTelemetry(ctx context.Context, filter models.TelemetryQueryFilter) (*models.TelemetryQueryResponse, error) {
	if filter.MMSI <= 0 {
		return nil, errors.New("el código MMSI no es válido")
	}
	if filter.From.After(filter.To) {
		return nil, errors.New("la fecha 'from' no puede ser posterior a 'to'")
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.GetShipTelemetry(ctx, filter)
}

// SearchShips realiza búsquedas facetadas sobre static_ais aplicando filtros y paginación
func (r *repository) SearchShips(ctx context.Context, filter models.ShipSearchFilter) (*models.ShipSearchResponse, error) {
	db := r.db.WithContext(ctx).Model(&models.DBStaticAIS{})

	if filter.Query != "" {
		likePattern := "%" + strings.ToLower(filter.Query) + "%"
		db = db.Where("LOWER(shipname) LIKE LOWER(?)", likePattern)
	}

	if filter.IMO != nil {
		db = db.Where("imo = ?", *filter.IMO)
	}

	if filter.MMSI != nil {
		db = db.Where("mmsi = ?", *filter.MMSI)
	}

	if filter.ShipType != "" {
		db = db.Where("LOWER(ship_type) = LOWER(?)", filter.ShipType)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var dbShips []models.DBStaticAIS
	if err := db.Limit(limit).Offset(offset).Find(&dbShips).Error; err != nil {
		return nil, err
	}

	ships := make([]models.StaticAIS, len(dbShips))
	for i, s := range dbShips {
		ships[i] = *mapToDomain(&s)
	}

	return &models.ShipSearchResponse{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Ships:  ships,
	}, nil
}

// GetShipTelemetry recupera la telemetría dinámica acotada por MMSI y rango de fechas
func (r *repository) GetShipTelemetry(ctx context.Context, filter models.TelemetryQueryFilter) (*models.TelemetryQueryResponse, error) {
	var exists int
	res := r.db.WithContext(ctx).Raw("SELECT 1 FROM static_ais WHERE mmsi = ? LIMIT 1", filter.MMSI).Scan(&exists)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 || exists != 1 {
		return nil, models.ErrShipNotFound
	}

	query := r.db.WithContext(ctx).Model(&models.DBDynamicAIS{}).
		Where("mmsi = ? AND timestamp >= ? AND timestamp <= ?", filter.MMSI, filter.From, filter.To)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var dbTelemetry []models.DBDynamicAIS
	if err := query.Order("timestamp ASC").Limit(limit).Offset(offset).Find(&dbTelemetry).Error; err != nil {
		return nil, err
	}

	telemetry := make([]models.DynamicHistoryResponse, len(dbTelemetry))
	for i, d := range dbTelemetry {
		telemetry[i] = models.DynamicHistoryResponse{
			Timestamp: d.Timestamp,
			Latitude:  d.Latitude,
			Longitude: d.Longitude,
			Status:    d.Status,
			Speed:     d.Speed,
			Course:    d.Course,
		}
	}

	return &models.TelemetryQueryResponse{
		MMSI:      filter.MMSI,
		From:      filter.From,
		To:        filter.To,
		Total:     total,
		Telemetry: telemetry,
	}, nil
}
