package ais

import "time"

// ===========================================================
// MODELOS DE LA ENTIDAD AIS
// ===========================================================

// StaticAIS representa la tabla para mensajes AIS estaticos
type StaticAIS struct {
	ID        uint
	MsgType   int    `json:"msg_type"`
	IMO       int    `json:"imo"`
	MMSI      int    `json:"mmsi"`
	Callsign  string `json:"callsign"`
	Shipname  string `json:"shipname"`
	ShipType  string `json:"ship_type"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Dynamics  []DynamicAIS
}

// DynamicAIS representa la tabla para mensajes AIS dinámicos
type DynamicAIS struct {
	ID        uint
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
	CreatedAt time.Time
	UpdatedAt time.Time
}
