package repository

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/YESUBZERO/consumer-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// shipRepository maneja las operaciones de la base de datos
type shipRepository struct {
	db *gorm.DB
}

// NewShipRepository crea una nueva instancia del repositorio
func NewShipRepository(db *gorm.DB) *shipRepository {
	return &shipRepository{db: db}
}

// InitDB establece la conexión con la base de datos
// y crea la tabla si no existe
func InitDB(dsn string) (*gorm.DB, error) {

	// Configurar logger personalizado
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,  // Umbral de tiempo para considerar una consulta lenta
			LogLevel:                  logger.Error, // Nivel de log para ocultar logs de info
			IgnoreRecordNotFoundError: true,         // Ignorar errores de registro no encontrado
			Colorful:                  true,         // Habilitar colores en el log
		},
	)

	// Inicializar la base de datos
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, err
	}

	// Configurar pool de conexiones
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	// Auto-migrar el esquema de la base de datos
	err = db.AutoMigrate(&models.Ship{}, &models.ShipPosition{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// IMOExists verifica si un barco ya está almacenado en la BD
func (r *shipRepository) IMOExists(imo int) (bool, error) {
	var exists bool
	// Verificar si el barco existe
	err := r.db.Model(&models.Ship{}).
		Select("count(1) > 0").
		Where("imo = ?", imo).
		Scan(&exists).Error

	return exists, err
}

// MMSIExists verifica si un barco ya está almacenado en la BD
func (r *shipRepository) MMSIExists(mmsi int) (bool, error) {
	var exists bool
	// Verificar si el barco existe
	err := r.db.Model(&models.Ship{}).
		Select("count(1) > 0").
		Where("mmsi = ?", mmsi).
		Scan(&exists).Error

	return exists, err
}

// SaveShip guarda el barco en la base de datos
func (r *shipRepository) SaveShip(ship *models.Ship) error {
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "imo"}},
		DoNothing: true,
	}).Create(ship).Error; err != nil {
		return fmt.Errorf("Error guardando buque en la base de datos: %w", err)
	}
	return nil
}

// SavePosition guarda la posición del barco en la base de datos
func (r *shipRepository) SavePosition(pos *models.ShipPosition) error {
	return r.db.Create(pos).Error
}
