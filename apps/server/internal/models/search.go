package models

import "time"

// ShipSearchFilter define los criterios facetados y parámetros de paginación para búsqueda de buques
type ShipSearchFilter struct {
	Query    string `form:"q" json:"q"`
	IMO      *int   `form:"imo" json:"imo"`
	MMSI     *int   `form:"mmsi" json:"mmsi"`
	ShipType string `form:"ship_type" json:"ship_type"`
	Limit    int    `form:"limit" json:"limit"`
	Offset   int    `form:"offset" json:"offset"`
}

// ShipSearchResponse representa la respuesta paginada de búsqueda de buques
type ShipSearchResponse struct {
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Ships  []StaticAIS `json:"ships"`
}

// TelemetryQueryFilter define los criterios para acotar el historial de telemetría por ventana temporal
type TelemetryQueryFilter struct {
	MMSI   int       `form:"-" json:"mmsi"`
	From   time.Time `form:"from" json:"from"`
	To     time.Time `form:"to" json:"to"`
	Limit  int       `form:"limit" json:"limit"`
	Offset int       `form:"offset" json:"offset"`
}

// TelemetryQueryResponse representa la respuesta con telemetría acotada por rango de fechas
type TelemetryQueryResponse struct {
	MMSI      int                      `json:"mmsi"`
	From      time.Time                `json:"from"`
	To        time.Time                `json:"to"`
	Total     int64                    `json:"total"`
	Telemetry []DynamicHistoryResponse `json:"telemetry"`
}
