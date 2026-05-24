package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Aquí se implementa la conexión a PostgreSQL usando GORM
func InitPostgres(dsn string) (*gorm.DB, error) {

	// 1. Configuración de GORM para PostgreSQL
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,          // Don't include params in the SQL log
			Colorful:                  false,         // Disable color
		},
	)

	// 2. Abrir la conexión a PostgreSQL con GORM
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("error conectando a PostgreSQL: %w", err)
	}

	// 3. Configurar el pool de conexiones (performance y estabilidad)
	// Extraemos la isntancia de sql.DB para configurar el pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error al obtener la instancia SQL nativa: %w", err)
	}

	// Establece el numero maximo de conexiones en el pool de reservas (Inactivas)
	sqlDB.SetMaxIdleConns(25)

	// Establecer el número máximo de conexiones abiertas
	sqlDB.SetMaxOpenConns(25)

	// Establecer el tiempo máximo de vida de una conexión
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("✅ [MAGI-CORE] Conexión a PostgreSQL establecida con éxito y pool configurado.")
	return db, nil
}
