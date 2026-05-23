package ais

import (
	"time"

	"gorm.io/gorm"
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

// DBDynamicAIS representa la tabla para mensajes AIS dinámicos
type DBDynamicAIS struct {
	ID        uint `gorm:"primaryKey"`
	MsgType   int
	Timestamp string
	MMSI      int `gorm:"index"`
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

// ShipScraper representa la estructura almacenada en PostgreSQL
// type ShipScraper struct {
// 	IMO            int     `gorm:"primaryKey" json:"imo"`
// 	MMSI           int     `gorm:"uniqueIndex" json:"mmsi"`
// 	Callsign       string  `json:"callsign"`
// 	Shipname       string  `json:"shipname"`
// 	ShipType       string  `json:"ship_type"`
// 	BuiltYear      *string `gorm:"column:built_year" json:"Built"`
// 	Shipyard       *string `gorm:"column:shipyard" json:"shipyard"`
// 	HullNumber     *string `gorm:"column:hull_number" json:"Hull-No."`
// 	KeelLaying     *string `gorm:"column:keel_laying" json:"Keel Laying"`
// 	LaunchDate     *string `gorm:"column:launch_date" json:"Launch"`
// 	DeliveryDate   *string `gorm:"column:delivery_date" json:"Delivery"`
// 	GT             *string `gorm:"column:gt" json:"gt"`
// 	NT             *string `gorm:"column:nt" json:"nt"`
// 	CarryingCapTDW *string `gorm:"column:carrying_capacity_tdw" json:"Carrying capacity (tdw)"`
// 	LengthOverall  *string `gorm:"column:length_overall" json:"Length overall (m)"`
// 	Breadth        *string `gorm:"column:breadth" json:"Breadth (m)"`
// 	Depth          *string `gorm:"column:depth" json:"Depth (m)"`
// 	Propulsion     *string `gorm:"column:propulsion" json:"propulsion"`
// 	Power          *string `gorm:"column:power" json:"power"`
// 	Screws         *string `gorm:"column:screws" json:"screws"`
// 	Speed          *string `gorm:"column:speed" json:"speed"`
// }

// ===========================================================
// INTERFAZ Y CONTRATO DEL REPOSITORIO
// ===========================================================

// Repository define los metodos para las capas superiores
type Repository interface {
	SaveStatic(ais *StaticAIS) error
	SaveDynamic(ais *DynamicAIS) error
	GetByIMO(imo int) (*StaticAIS, error)
	GetByMMSI(mmsi int) (*StaticAIS, error)
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

	// 1. Comprobamos si existe el registro estatico del buque
	var count int64
	err := r.db.Model(&DBStaticAIS{}).
		Where("mmsi = ?", ais.MMSI).
		Count(&count).Error

	if err != nil {
		return err
	}

	// 2. Si el buque no existe creamos el registro momentaneo
	if count == 0 {
		staticMessage := DBStaticAIS{
			MMSI:     ais.MMSI,
			Shipname: "UNKNOWN", // Se actualiza cuando llegue el mensaje estatico
		}
		if error := r.db.Create(&staticMessage).Error; error != nil {
			return error
		}
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

	// 4. Retornamos la consulta
	return r.db.Create(&dynamicMessage).Error
}

// GetByIMO busca un buque estatico e incluye todo su historial de posiciones.
func (r *repository) GetByIMO(imo int) (*StaticAIS, error) {
	// 1. Cargamos la relacion de Dynamics usando FK con Preload
	var staticMessage DBStaticAIS
	err := r.db.Preload("Dynamics").
		Where("imo = ?", imo).
		First(&staticMessage).Error

	if err != nil {
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

// GetByMMSI obtiene informacion de un barco por MMSI
func (r *repository) GetByMMSI(mmsi int) (*StaticAIS, error) {
	// método para obtener informacion dinamica de un barco
	return nil, nil
}
