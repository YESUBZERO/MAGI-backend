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

// Ship representa la estructura almacenada en PostgreSQL
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

// SaveStatic guarda el barco en la base de datos
func (db *repository) SaveStatic(ais *StaticAIS) error {
	// método para guardar barcos (mensajes estaticos)
	return nil
}

// SaveDynamic guarda la posición del barco en la base de datos
func (db *repository) SaveDynamic(ais *DynamicAIS) error {
	// método para guardar mensajes dinamicos
	return nil
}

// GetByIMO obtiene informacion de un buque por IMO
func (db *repository) GetByIMO(imo int) (*StaticAIS, error) {
	// método para obtener información de un barco
	return nil, nil
}

// GetByMMSI obtiene informacion de un
