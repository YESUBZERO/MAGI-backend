package ais

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ===========================================================
// MODELOS DE INFRAESTRUCTURA DE LA ENTIDAD AIS
// ===========================================================

// DBStaticAIS representa la tabla para mensajes AIS estaticos
type DBStaticAIS struct {
	ID        uint `gorm:"primaryKey"`
	MsgType   int
	IMO       int
	MMSI      int `gorm:"uniqueIndex"`
	Callsign  string
	Shipname  string
	ShipType  string
	CreatedAt time.Time
	UpdatedAt time.Time
	Dynamics  []DBDynamicAIS `gorm:"foreignKey:MMSI;references:MMSI"`
}

// DBDynamicAIS representa la tabla para mensajes AIS dinámicos.
// El índice único compuesto (MMSI, Timestamp) garantiza idempotencia:
// si Kafka reenvía un mensaje ya procesado, el INSERT se ignora sin error.
type DBDynamicAIS struct {
	ID        uint `gorm:"primaryKey"`
	MsgType   int
	Timestamp time.Time `gorm:"uniqueIndex:idx_dynamic_mmsi_timestamp"`
	MMSI      int       `gorm:"uniqueIndex:idx_dynamic_mmsi_timestamp"`
	Status    string
	Turn      float64
	Speed     float64
	Accuracy  bool
	Longitude float64
	Latitude  float64
	Course    float64
	Heading   int
	Second    int
	Maneuver  int
	Raim      bool
	Radio     int
	CreatedAt time.Time
	UpdatedAt time.Time
}



// ===========================================================
// INTERFAZ Y CONTRATO DEL REPOSITORIO
// ===========================================================

// Repository define los metodos para las capas superiores
type Repository interface {
	SaveStatic(ais *StaticAIS) error
	SaveDynamic(ais *DynamicAIS) error
	GetByIMO(imo int, limit, offset int) (*StaticAIS, error)
	GetByMMSI(mmsi int, limit, offset int) (*StaticAIS, error)
}

// Definimos un struct type para implementar
type repository struct {
	db *gorm.DB
}

// Crear el constructor del repositorio
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// ===========================================================
// IMPLEMENTACION DE MÉTODOS DE BASE DE DATOS
// ===========================================================

// SaveStatic guarda o actualiza la informacion del barco
func (r *repository) SaveStatic(ais *StaticAIS) error {
	// método para guardar barcos (mensajes estaticos)
	staticMessage := DBStaticAIS{
		MsgType:  ais.MsgType,
		IMO:      ais.IMO,
		MMSI:     ais.MMSI,
		Callsign: ais.Callsign,
		Shipname: ais.Shipname,
		ShipType: ais.ShipType,
	}
	return r.db.Where(DBStaticAIS{MMSI: ais.MMSI}).
		Assign(staticMessage).
		FirstOrCreate(&staticMessage).Error
}

// SaveDynamic guarda la posición del barco
func (r *repository) SaveDynamic(ais *DynamicAIS) error {

	// 1. Garantizamos la existencia del buque de forma atómica e idempotente.
	//    Si ya existe, OnConflict ignora la inserción sin error y sin colisiones de concurrencia.
	placeholder := DBStaticAIS{
		MMSI:     ais.MMSI,
		Shipname: "UNKNOWN", // Se actualizará cuando llegue el mensaje estático
	}
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "mmsi"}},
		DoNothing: true,
	}).Create(&placeholder).Error; err != nil {
		return err
	}

	// 3. Mapeamos el dominio al modelo GORM
	dynamicMessage := DBDynamicAIS{
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

	// 4. Insertamos con OnConflict DoNothing para garantizar idempotencia.
	//    Si Kafka reenvía un mensaje duplicado (mismo MMSI + Timestamp),
	//    el INSERT se ignora silenciosamente sin retornar error.
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&dynamicMessage).Error
}

// GetByIMO busca un buque estatico e incluye su historial de posiciones paginado.
func (r *repository) GetByIMO(imo int, limit, offset int) (*StaticAIS, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	// 1. Cargamos la relacion de Dynamics usando FK con Preload ordenado y limitado
	var staticMessage DBStaticAIS
	err := r.db.Preload("Dynamics", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp DESC").Limit(limit).Offset(offset)
	}).Where("imo = ?", imo).First(&staticMessage).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShipNotFound
		}
		return nil, err
	}

	// 2. Mapeamos de vuelta al modelo de dominio limpio
	dynamicMessages := make([]DynamicAIS, len(staticMessage.Dynamics))
	for i, d := range staticMessage.Dynamics {
		dynamicMessages[i] = DynamicAIS{
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

	// 3. Retornamos el modelo
	return &StaticAIS{
		ID:        staticMessage.ID,
		MsgType:   staticMessage.MsgType,
		IMO:       staticMessage.IMO,
		MMSI:      staticMessage.MMSI,
		Callsign:  staticMessage.Callsign,
		Shipname:  staticMessage.Shipname,
		ShipType:  staticMessage.ShipType,
		CreatedAt: staticMessage.CreatedAt,
		UpdatedAt: staticMessage.UpdatedAt,
		Dynamics:  dynamicMessages,
	}, nil
}

// GetByMMSI obtiene información de un barco por su código MMSI con telemetría paginada.
func (r *repository) GetByMMSI(mmsi int, limit, offset int) (*StaticAIS, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	var staticMessage DBStaticAIS
	err := r.db.Preload("Dynamics", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp DESC").Limit(limit).Offset(offset)
	}).Where("mmsi = ?", mmsi).First(&staticMessage).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShipNotFound
		}
		return nil, err
	}

	dynamicMessages := make([]DynamicAIS, len(staticMessage.Dynamics))
	for i, d := range staticMessage.Dynamics {
		dynamicMessages[i] = DynamicAIS{
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

	return &StaticAIS{
		ID:        staticMessage.ID,
		MsgType:   staticMessage.MsgType,
		IMO:       staticMessage.IMO,
		MMSI:      staticMessage.MMSI,
		Callsign:  staticMessage.Callsign,
		Shipname:  staticMessage.Shipname,
		ShipType:  staticMessage.ShipType,
		CreatedAt: staticMessage.CreatedAt,
		UpdatedAt: staticMessage.UpdatedAt,
		Dynamics:  dynamicMessages,
	}, nil
}
