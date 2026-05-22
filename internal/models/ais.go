package models

// AISMessageType5 representa los datos AIS estáticos
type Ship struct {
	MsgType  int    `json:"msg_type"`
	IMO      int    `gorm:"primaryKey" json:"imo"`
	MMSI     int    `gorm:"uniqueIndex" json:"mmsi"`
	Callsign string `json:"callsign"`
	Shipname string `json:"shipname"`
	ShipType string `json:"ship_type"`
}

// ShipPosition representa la posición dinámica de un barco sin un ID autoincremental
type ShipPosition struct {
	Timestamp string  `gorm:"primaryKey" json:"timestamp"`
	MMSI      int     `gorm:"primaryKey;constraint:OnDelete:CASCADE;" json:"mmsi"`
	Status    string  `json:"status"`
	Turn      float64 `json:"turn"`
	Speed     float64 `json:"speed"`
	Accuracy  bool    `json:"accuracy"`
	Longitude float64 `json:"lon"`
	Latitude  float64 `json:"lat"`
	Course    float64 `json:"course"`
	Heading   int     `json:"heading"`
	Second    int     `json:"second"`
	Maneuver  int     `json:"maneuver"`
	Raim      bool    `json:"raim"`
	Radio     int     `json:"radio"`
}
