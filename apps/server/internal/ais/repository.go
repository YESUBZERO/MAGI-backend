package ais

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aires-magi/internal/models"
)

// Repository define el contrato de acceso a datos para AIS
type Repository interface {
	SaveStatic(ais *models.StaticAIS) error
	SaveDynamic(ais *models.DynamicAIS) error
	GetByIMO(imo int, limit, offset int) (*models.StaticAIS, error)
	GetByMMSI(mmsi int, limit, offset int) (*models.StaticAIS, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository construye el repositorio AIS con la conexión GORM
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// SaveStatic guarda o actualiza la información estática del buque
func (r *repository) SaveStatic(ais *models.StaticAIS) error {
	staticMessage := models.DBStaticAIS{
		MsgType:  ais.MsgType,
		IMO:      ais.IMO,
		MMSI:     ais.MMSI,
		Callsign: ais.Callsign,
		Shipname: ais.Shipname,
		ShipType: ais.ShipType,
	}

	return r.db.Where(models.DBStaticAIS{MMSI: ais.MMSI}).
		Assign(staticMessage).
		FirstOrCreate(&staticMessage).Error
}

// SaveDynamic guarda la posición del buque garantizando idempotencia (at-least-once)
func (r *repository) SaveDynamic(ais *models.DynamicAIS) error {
	// 1. Asegurar existencia del buque de forma atómica e idempotente
	placeholder := models.DBStaticAIS{
		MMSI:     ais.MMSI,
		Shipname: "UNKNOWN",
	}
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "mmsi"}},
		DoNothing: true,
	}).Create(&placeholder).Error; err != nil {
		return err
	}

	// 2. Mapear a modelo GORM
	dynamicMessage := models.DBDynamicAIS{
		MsgType:   ais.MsgType,
		Timestamp: ais.Timestamp,
		MMSI:      ais.MMSI,
		Status:    ais.Status,
		Turn:      ais.Turn,
		Speed:     ais.Speed,
		Accuracy:  ais.Accuracy,
		Longitude: ais.Longitude,
		Latitude:  ais.Latitude,
		Course:    ais.Course,
		Heading:   ais.Heading,
		Second:    ais.Second,
		Maneuver:  ais.Maneuver,
		Raim:      ais.Raim,
		Radio:     ais.Radio,
	}

	// 3. OnConflict DoNothing para ignorar duplicados (MMSI + Timestamp) sin error
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&dynamicMessage).Error
}

// GetByIMO busca un buque por su número IMO con historial paginado
func (r *repository) GetByIMO(imo int, limit, offset int) (*models.StaticAIS, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	var staticMessage models.DBStaticAIS
	err := r.db.Preload("Dynamics", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp DESC").Limit(limit).Offset(offset)
	}).Where("imo = ?", imo).First(&staticMessage).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrShipNotFound
		}
		return nil, err
	}

	return mapToDomain(&staticMessage), nil
}

// GetByMMSI busca un buque por su código MMSI con telemetría paginada
func (r *repository) GetByMMSI(mmsi int, limit, offset int) (*models.StaticAIS, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	var staticMessage models.DBStaticAIS
	err := r.db.Preload("Dynamics", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp DESC").Limit(limit).Offset(offset)
	}).Where("mmsi = ?", mmsi).First(&staticMessage).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrShipNotFound
		}
		return nil, err
	}

	return mapToDomain(&staticMessage), nil
}

func mapToDomain(dbMsg *models.DBStaticAIS) *models.StaticAIS {
	dynamicMessages := make([]models.DynamicAIS, len(dbMsg.Dynamics))
	for i, d := range dbMsg.Dynamics {
		dynamicMessages[i] = models.DynamicAIS{
			ID:        d.ID,
			MsgType:   d.MsgType,
			Timestamp: d.Timestamp,
			MMSI:      d.MMSI,
			Status:    d.Status,
			Turn:      d.Turn,
			Speed:     d.Speed,
			Accuracy:  d.Accuracy,
			Longitude: d.Longitude,
			Latitude:  d.Latitude,
			Course:    d.Course,
			Heading:   d.Heading,
			Second:    d.Second,
			Maneuver:  d.Maneuver,
			Raim:      d.Raim,
			Radio:     d.Radio,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}

	return &models.StaticAIS{
		ID:        dbMsg.ID,
		MsgType:   dbMsg.MsgType,
		IMO:       dbMsg.IMO,
		MMSI:      dbMsg.MMSI,
		Callsign:  dbMsg.Callsign,
		Shipname:  dbMsg.Shipname,
		ShipType:  dbMsg.ShipType,
		CreatedAt: dbMsg.CreatedAt,
		UpdatedAt: dbMsg.UpdatedAt,
		Dynamics:  dynamicMessages,
	}
}
