<div align="center">

# MAGI · consumer-service

**Microservicio de ingesta de datos AIS en tiempo real**

*Parte del sistema [MAGI — Maritime AIS Global Intelligence]*

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go&logoColor=white)
![Kafka](https://img.shields.io/badge/Kafka-Consumer_Group-231F20?style=flat-square&logo=apachekafka&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-GORM-4169E1?style=flat-square&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-multi--stage-2496ED?style=flat-square&logo=docker&logoColor=white)

</div>

---

## ¿Qué hace este servicio?

`consumer-service` escucha dos tópicos de Apache Kafka con datos del protocolo **AIS** (*Automatic Identification System*) — el estándar internacional de seguimiento de embarcaciones —, los valida, los persiste en PostgreSQL y expone una REST API para consultarlos.

| Tópico Kafka | Tipo de mensaje | Contenido |
|---|---|---|
| `KAFKA_STATIC_TOPIC` | Estático (AIS Tipo 5) | Identidad del buque: IMO, MMSI, nombre, tipo, indicativo |
| `KAFKA_DYNAMIC_TOPIC` | Dinámico (AIS Tipo 1/2/3) | Telemetría en tiempo real: posición GPS, velocidad, rumbo |

---

## Arquitectura

El servicio implementa una arquitectura de **capas limpias** donde cada capa solo conoce la interfaz de la inferior, nunca su implementación concreta.

```
┌─────────────────────────────────────────────────────────────────────┐
│                        consumer-service                             │
│                                                                     │
│   Kafka Broker                                                      │
│   ┌──────────┐    ┌─────────────────────────────────────────────┐   │
│   │ ais.     │    │  kafka.ConsumerGroup                        │   │
│   │ static   │───▶│                                             │   │
│   ├──────────┤    │  ConsumeClaim() ──▶ chan *kafkaJob (x500)   │   │
│   │ ais.     │    │                           │                 │   │
│   │ dynamic  │───▶│  Worker Pool (x5) ◀───────┘                 │   │
│   └──────────┘    └──────────────┬──────────────────────────────┘   │
│                                  │                                  │
│                    ┌─────────────▼──────────────┐                   │
│   HTTP Client      │       ais.Service          │                   │
│   ┌──────────┐     │  (validaciones de dominio) │                   │
│   │ REST API │────▶│                            │                   │
│   └──────────┘     └─────────────┬──────────────┘                   │
│                                  │                                  │
│                    ┌─────────────▼──────────────┐                   │
│                    │      ais.Repository        │                   │
│                    │  (GORM · PostgreSQL)       │                   │
│                    └────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Garantía de entrega — *at-least-once*

El offset de Kafka **solo se commitea después** de que el dato ha sido persistido en PostgreSQL. Si el proceso cae en medio de una escritura, Kafka reenviará el mensaje al reiniciar. Para evitar duplicados ante reenvíos, `SaveDynamic` usa `INSERT ... ON CONFLICT DO NOTHING` sobre un índice único compuesto `(MMSI, Timestamp)`.

---

## Stack tecnológico

| Dependencia | Versión | Rol |
|---|---|---|
| [IBM/sarama](https://github.com/IBM/sarama) | v1.45.0 | Cliente Kafka — Consumer Group |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | v1.12.0 | Framework HTTP |
| [gorm.io/gorm](https://gorm.io) | v1.25.12 | ORM — PostgreSQL |
| [kelseyhightower/envconfig](https://github.com/kelseyhightower/envconfig) | v1.4.0 | Configuración por variables de entorno |

---

## Estructura del proyecto

```
consumer-service/
├── cmd/
│   └── main.go                  # Wiring de dependencias y arranque del proceso
│
├── internal/
│   ├── ais/
│   │   ├── ais.go               # Modelos de dominio: StaticAIS, DynamicAIS
│   │   ├── handler.go           # Controlador HTTP: DTOs de request/response + rutas
│   │   ├── repository.go        # Capa de datos: modelos GORM + consultas PostgreSQL
│   │   └── service.go           # Lógica de negocio y validaciones de dominio
│   │
│   ├── config/
│   │   └── config.go            # Carga y validación de variables de entorno
│   │
│   ├── database/
│   │   └── postgres.go          # Inicialización de la conexión y pool de conexiones
│   │
│   └── kafka/
│       └── consumer.go          # Consumer Group · worker pool · at-least-once delivery
│      
│
├── Dockerfile                   # Build multi-stage: builder (golang:alpine) + runtime (alpine)
├── go.mod
└── go.sum
```

---

## Configuración

El servicio se configura **exclusivamente por variables de entorno**. No se requiere ningún archivo de configuración en producción.

| Variable | Requerida | Descripción | Ejemplo |
|---|---|---|---|
| `KAFKA_BROKERS` | ✅ | Lista de brokers (separados por coma) | `kafka:9092` |
| `KAFKA_STATIC_TOPIC` | ✅ | Tópico de mensajes AIS estáticos | `ais.static` |
| `KAFKA_DYNAMIC_TOPIC` | ✅ | Tópico de mensajes AIS dinámicos | `ais.dynamic` |
| `KAFKA_GROUP_ID` | ✅ | Consumer Group ID | `magi-consumer-group` |
| `DATABASE_DSN` | ✅ | DSN de conexión a PostgreSQL | `host=db user=magi password=secret dbname=magi port=5432 sslmode=disable` |

> Para desarrollo local, crea un archivo `.env` en la raíz (ya excluido en `.gitignore`) y expórtalo antes de ejecutar el binario.

### Ejemplo `.env`

```env
KAFKA_BROKERS=localhost:9092
KAFKA_STATIC_TOPIC=ais.static
KAFKA_DYNAMIC_TOPIC=ais.dynamic
KAFKA_GROUP_ID=magi-consumer-group
DATABASE_DSN=host=localhost user=magi password=secret dbname=magi port=5432 sslmode=disable
```

---

## Ejecución local

### Prerrequisitos

- Go 1.23+
- PostgreSQL corriendo y accesible
- Apache Kafka con los tópicos `ais.static` y `ais.dynamic` creados

```bash
# Clonar el repositorio
git clone https://github.com/YESUBZERO/MAGI-consumer.git
cd MAGI-consumer

# Exportar variables de entorno (o usar un .env)
export $(cat .env | xargs)

# Descargar dependencias
go mod download

# Ejecutar
go run ./cmd/main.go
```

El servicio arrancará e imprimirá en consola:

```
✅ [MAGI-CORE] Conexión a PostgreSQL establecida con éxito y pool configurado.
🔄 Ejecutando migraciones de base de datos...
📡 [MAGI-CORE] Consumer Group 'magi-consumer-group' escuchando tópicos: [ais.static ais.dynamic]
🚀 [MAGI-CORE] Servidor HTTP escuchando en el puerto :8080
🚢 [MAGI-CORE] Todos los servicios en marcha. Presiona Ctrl+C para detener.
```

---

## Docker

### Build

```bash
docker build -t magi/consumer-service:latest .
```

El Dockerfile usa un **build multi-stage**: la imagen final es Alpine mínima (~10 MB) y el binario se ejecuta con un usuario sin privilegios (`appuser`).

### Run

```bash
docker run -d \
  --name magi-consumer \
  -p 8080:8080 \
  -e KAFKA_BROKERS=kafka:9092 \
  -e KAFKA_STATIC_TOPIC=ais.static \
  -e KAFKA_DYNAMIC_TOPIC=ais.dynamic \
  -e KAFKA_GROUP_ID=magi-consumer-group \
  -e DATABASE_DSN="host=db user=magi password=secret dbname=magi port=5432 sslmode=disable" \
  magi/consumer-service:latest
```

---

## API REST

**Base URL:** `http://localhost:8080/api/v1`

### `POST /static` — Registrar buque

Crea o actualiza los datos estáticos de un buque. Equivale al flujo que hace el consumer cuando recibe un mensaje AIS estático vía Kafka.

**Request**
```json
{
  "msg_type": 5,
  "imo": 9321483,
  "mmsi": 636092298,
  "callsign": "A8IG4",
  "shipname": "EVER GIVEN",
  "ship_type": "Container Ship"
}
```

**Response** `201 Created`
```json
{
  "message": "Datos estáticos del buque procesados correctamente"
}
```

---

### `GET /ship/:imo` — Consultar buque por IMO

Devuelve la información completa del buque y todo su historial de posiciones registradas.

**Parámetros de ruta**

| Param | Tipo | Descripción |
|---|---|---|
| `imo` | `int` | Número IMO del buque (7 dígitos) |

**Response** `200 OK`
```json
{
  "imo": 9321483,
  "mmsi": 636092298,
  "callsign": "A8IG4",
  "shipname": "EVER GIVEN",
  "ship_type": "Container Ship",
  "total_positions_recorded": 3,
  "position_history": [
    {
      "timestamp": "2024-03-21T07:04:00Z",
      "latitude": 30.0444,
      "longitude": 32.5498,
      "status": "Under way using engine",
      "speed_knots": 7.8,
      "course": 164.5
    }
  ]
}
```

**Errores**

| Código | Descripción |
|---|---|
| `400` | IMO no es un número válido |
| `404` | Buque no encontrado |

---

## Flujo de datos completo

```
Fuente AIS (NMEA / API externa)
          │
          ▼
    Kafka Producer
    ┌─────┴──────┐
    │            │
    ▼            ▼
ais.static   ais.dynamic
    │            │
    └─────┬──────┘
          ▼
  Consumer Group (KAFKA_GROUP_ID)
  ┌───────────────────────────────────┐
  │  ConsumeClaim()                   │
  │    └─▶ chan *kafkaJob (buf: 500)  │
  │              │                    │
  │    ┌─────────▼───────────┐        │
  │    │   Worker Pool (×5)  │        │
  │    └─────────┬───────────┘        │
  └─────────────┼────────────────────┘
                │
     ┌──────────┴──────────┐
     │                     │
     ▼                     ▼
 StaticAIS            DynamicAIS
     │                     │
     ▼                     ▼
SaveStatic()          SaveDynamic()
FirstOrCreate         INSERT ... ON CONFLICT DO NOTHING
(MMSI único)          (idx: MMSI + Timestamp)
     │                     │
     └──────────┬──────────┘
                ▼
          PostgreSQL
                │
                ▼
      session.MarkMessage()   ← offset commiteado solo si BD tuvo éxito
```
