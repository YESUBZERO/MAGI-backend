package models

import (
	"errors"
	"time"
)

// Errores de dominio marítimo
var (
	ErrShipNotFound = errors.New("buque no encontrado")
)

// ===========================================================
// MODELOS DE DOMINIO AIS
// ===========================================================

// StaticAIS representa los datos estáticos de identidad y viaje de un buque
type StaticAIS struct {
	ID        uint         `json:"id"`
	MsgType   int          `json:"msg_type"`
	IMO       int          `json:"imo"`
	MMSI      int          `json:"mmsi"`
	Callsign  string       `json:"callsign"`
	Shipname  string       `json:"shipname"`
	ShipType  string       `json:"ship_type"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Dynamics  []DynamicAIS `json:"dynamics,omitempty"`
}

// DynamicAIS representa la posición y telemetría dinámica de un buque en navegación
type DynamicAIS struct {
	ID        uint      `json:"id"`
	MsgType   int       `json:"msg_type"`
	Timestamp time.Time `json:"timestamp"`
	MMSI      int       `json:"mmsi"`
	Status    string    `json:"status"`
	Turn      float64   `json:"turn"`
	Speed     float64   `json:"speed"`
	Accuracy  bool      `json:"accuracy"`
	Longitude float64   `json:"lon"`
	Latitude  float64   `json:"lat"`
	Course    float64   `json:"course"`
	Heading   int       `json:"heading"`
	Second    int       `json:"second"`
	Maneuver  int       `json:"maneuver"`
	Raim      bool      `json:"raim"`
	Radio     int       `json:"radio"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ===========================================================
// MODELOS DE INFRAESTRUCTURA GORM
// ===========================================================

// DBStaticAIS representa la tabla relacional PostgreSQL para buques estáticos
type DBStaticAIS struct {
	ID        uint           `gorm:"primaryKey"`
	MsgType   int            `gorm:"column:msg_type"`
	IMO       int            `gorm:"column:imo;index"`
	MMSI      int            `gorm:"column:mmsi;uniqueIndex"`
	Callsign  string         `gorm:"column:callsign"`
	Shipname  string         `gorm:"column:shipname"`
	ShipType  string         `gorm:"column:ship_type"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	Dynamics  []DBDynamicAIS `gorm:"foreignKey:MMSI;references:MMSI"`
}

func (DBStaticAIS) TableName() string {
	return "static_ais"
}

// DBDynamicAIS representa la tabla relacional PostgreSQL para telemetría dinámica.
// El índice único compuesto (MMSI, Timestamp) garantiza idempotencia estricta en la ingesta.
type DBDynamicAIS struct {
	ID        uint      `gorm:"primaryKey"`
	MsgType   int       `gorm:"column:msg_type"`
	Timestamp time.Time `gorm:"column:timestamp;uniqueIndex:idx_dynamic_mmsi_timestamp"`
	MMSI      int       `gorm:"column:mmsi;uniqueIndex:idx_dynamic_mmsi_timestamp"`
	Status    string    `gorm:"column:status"`
	Turn      float64   `gorm:"column:turn"`
	Speed     float64   `gorm:"column:speed"`
	Accuracy  bool      `gorm:"column:accuracy"`
	Longitude float64   `gorm:"column:longitude"`
	Latitude  float64   `gorm:"column:latitude"`
	Course    float64   `gorm:"column:course"`
	Heading   int       `gorm:"column:heading"`
	Second    int       `gorm:"column:second"`
	Maneuver  int       `gorm:"column:maneuver"`
	Raim      bool      `gorm:"column:raim"`
	Radio     int       `gorm:"column:radio"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (DBDynamicAIS) TableName() string {
	return "dynamic_ais"
}

// ===========================================================
// DTOs (Data Transfer Objects) PARA API HTTP
// ===========================================================

// CreateStaticMessageRequest payload para registrar o actualizar buque
type CreateStaticMessageRequest struct {
	MsgType  int    `json:"msg_type" binding:"required"`
	IMO      int    `json:"imo" binding:"required"`
	MMSI     int    `json:"mmsi" binding:"required"`
	Callsign string `json:"callsign"`
	Shipname string `json:"shipname" binding:"required"`
	ShipType string `json:"ship_type"`
}

// ShipDetailsResponse respuesta enriquecida para el cliente HTTP
type ShipDetailsResponse struct {
	IMO        int                      `json:"imo"`
	MMSI       int                      `json:"mmsi"`
	Callsign   string                   `json:"callsign"`
	Shipname   string                   `json:"shipname"`
	ShipType   string                   `json:"ship_type"`
	TotalSpots int                      `json:"total_positions_recorded"`
	History    []DynamicHistoryResponse `json:"position_history,omitempty"`
}

// DynamicHistoryResponse histórico de un punto de telemetría
type DynamicHistoryResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Status    string    `json:"status"`
	Speed     float64   `json:"speed_knots"`
	Course    float64   `json:"course"`
}
