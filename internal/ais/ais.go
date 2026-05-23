package ais

import "time"

// ===========================================================
// MODELOS DE LA ENTIDAD AIS
// ===========================================================

// StaticAIS representa la tabla para mensajes AIS estaticos
type StaticAIS struct {
	ID        uint
	MsgType   int
	IMO       int
	MMSI      int
	Callsign  string
	Shipname  string
	ShipType  string
	CreatedAt time.Time
	UpdatedAt time.Time
	Dynamics  []DynamicAIS
}

// DynamicAIS representa la tabla para mensajes AIS dinámicos
type DynamicAIS struct {
	ID        uint
	MsgType   int
	Timestamp string
	MMSI      int
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
