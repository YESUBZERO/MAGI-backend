package ais

import "errors"

// ===========================================================
// INTERFAZ Y CONTRATO DEL SERVICIO
// ===========================================================

// 1. Definir el contrato Service del negocio ais
type Service interface {
	ProcessStaticMessage(ais *StaticAIS) error
	ProcessDynamicMessage(ais *DynamicAIS) error
	GetShipByIMO(imo int) (*StaticAIS, error)
}

// 2. Definir la implementacion concreta de la interfaz
type service struct {
	repo Repository
}

// ===========================================================
// IMPLEMENTACION DE MÉTODOS DEL SERVICIO
// ===========================================================

// 3. Definir el constructor para inyectar el repositorio
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// 4. Definir el método ProcessStaticMessage para mensajes estaticos.
func (s *service) ProcessStaticMessage(msg *StaticAIS) error {
	// 1. El MMSI es el identificador marítimo obligatorio
	if msg.MMSI <= 0 {
		return errors.New("el número MMSI es obligatorio")
	}

	// 2. El IMO debe ser un número válido de 7 digitos
	if msg.IMO < 1000000 || msg.IMO > 9999999 {
		return errors.New("el número IMO no es válido")
	}

	// 3. Asegurar que el nombre del barco no guarde espacios vacios
	if msg.Shipname == "" {
		msg.Shipname = "UNKNOWN"
	}

	// 4. Enviamos al repositorio tras las validaciones.
	return s.repo.SaveStatic(msg)
}

// 5. Definir el método ProcessDynamicMessage para mensajes dinamicos
func (s *service) ProcessDynamicMessage(msg *DynamicAIS) error {
	// 1. Validar el identificador MMSI del buque
	if msg.MMSI <= 0 {
		return errors.New("el código MMSI es obligatorio para registrar telemetría")
	}

	// 2.Validar el rango geográfico, evitar coordenadas corruptas

	// 3. La velocidad de un barco en nudos no puede ser negativa
	if msg.Speed < 0 {
		msg.Speed = 0.0
	}

	// 4. Enviamos al repositorio
	return s.repo.SaveDynamic(msg)
}

// 6. Definir el método GetShipByIMO para obtener info de un buque
func (s *service) GetShipByIMO(imo int) (*StaticAIS, error) {
	// 1. El numero IMO proporcionado debe ser valido
	if imo < 1000000 || imo > 9999999 {
		return nil, errors.New("el número IMO no es válido")
	}

	return s.repo.GetByIMO(imo)
}
