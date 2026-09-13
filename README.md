<div align="center">

# MAGI · backend

**Microservicio de ingesta de datos AIS en tiempo real**

*Parte del sistema [MAGI — Monitoreo y Análisis de Gases e Inmisiones]*

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go&logoColor=white)
![Kafka](https://img.shields.io/badge/Kafka-Consumer_Group-231F20?style=flat-square&logo=apachekafka&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-GORM-4169E1?style=flat-square&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-multi--stage-2496ED?style=flat-square&logo=docker&logoColor=white)

</div>

---

## ¿Qué hace este servicio?

`MAGI - backend` escucha dos tópicos de Apache Kafka con datos del protocolo **AIS** (*Automatic Identification System*) — el estándar internacional de seguimiento de embarcaciones —, los valida, los persiste en PostgreSQL y expone una REST API para consultarlos.

| Tópico Kafka | Tipo de mensaje | Contenido |
|---|---|---|
| `KAFKA_STATIC_TOPIC` | Estático (AIS Tipo 5) | Identidad del buque: IMO, MMSI, nombre, tipo, indicativo |
| `KAFKA_DYNAMIC_TOPIC` | Dinámico (AIS Tipo 1/2/3) | Telemetría en tiempo real: posición GPS, velocidad, rumbo |

---

## Arquitectura y Segregación CQRS

El sistema adopta el patrón **CQRS (*Command Query Responsibility Segregation*)**, desacoplando el flujo de ingesta del flujo de consulta en dos microservicios independientes que comparten los paquetes internos de persistencia y dominio:

1. **Microservicio Ingestor (`magi-worker` · `cmd/worker`):** Ingesta continua, validaciones de reglas marítimas (ITU-R M.1371) y persistencia atómica en PostgreSQL. No expone puertos web públicos.
2. **Microservicio de Consultas (`magi-api` · `cmd/api`):** Servidor HTTP REST (Gin) con paginación defensiva, telemetría histórica y endpoint de salud `/health`. No se conecta a Kafka.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                             MAGI - BACKEND (CQRS)                           │
│                                                                             │
│  [ WRITE PATH: magi-worker ]                                                │
│  Kafka Topics                                                               │
│  ┌──────────┐      ┌──────────────────────────────────────────────┐         │
│  │ais.static│───┐  │  kafka.ConsumerGroup                         │         │
│  ├──────────┤   │  │    ConsumeClaim() ──▶ chan *kafkaJob (x500)  │         │
│  │ais.dynam │───┼─▶│                             │                │         │
│  └──────────┘   │  │    Worker Pool (x5) ◀───────┘                │         │
│                 │  └──────────────┬───────────────────────────────┘         │
│                 │                 ▼                                         │
│                 │       ais.IngestionService                                │
│                 │                 │                                         │
│                 │                 ▼                                         │
│                 │           ais.Repository (Write)                          │
│                 │                 │ (INSERT ON CONFLICT DO NOTHING)         │
│                 │                 ▼                                         │
│                 │          [( PostgreSQL )]                                 │
│                 │                 ▲                                         │
│  [ READ PATH: magi-api ]          │                                         │
│  HTTP Clients                     │ (SELECT ORDER BY timestamp DESC LIMIT)  │
│  ┌──────────┐                     │                                         │
│  │ REST API │──▶ Gin Server ──▶ ais.QueryService ──▶ ais.Repository (Read)  │
│  └──────────┘     (:8080)                                                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Garantía de entrega — *at-least-once*

El offset de Kafka **solo se commitea después** de que el dato ha sido persistido en PostgreSQL. Si el proceso cae en medio de una escritura, Kafka reenviará el mensaje al reiniciar. Para evitar duplicados ante reenvíos, `SaveDynamic` usa `INSERT ... ON CONFLICT DO NOTHING` sobre un índice único compuesto `(MMSI, Timestamp)`.

---

## Stack tecnológico

| Dependencia | Versión | Rol |
|---|---|---|
| [IBM/sarama](https://github.com/IBM/sarama) | v1.45.0 | Cliente Kafka — Consumer Group |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | v1.12.0 | Framework HTTP para API REST |
| [gorm.io/gorm](https://gorm.io) | v1.25.12 | ORM — PostgreSQL |
| [kelseyhightower/envconfig](https://github.com/kelseyhightower/envconfig) | v1.4.0 | Configuración por variables de entorno |

---

## Estructura del proyecto

```
consumer-service/
├── cmd/
│   ├── api/
│   │   └── main.go              # Microservicio de Consultas HTTP REST (Read Path)
│   ├── worker/
│   │   └── main.go              # Microservicio Ingestor de Kafka (Write Path)
│   └── main.go                  # Entrypoint monolítico unificado (desarrollo local)
│
├── internal/
│   ├── ais/
│   │   ├── ais.go               # Modelos de dominio y errores centinela (ErrShipNotFound)
│   │   ├── handler.go           # Controlador HTTP: DTOs, validaciones y rutas
│   │   ├── repository.go        # Persistencia GORM con inserción atómica y paginación
│   │   └── service.go           # Interfaces segregadas: IngestionService y QueryService
│   │
│   ├── config/
│   │   └── config.go            # Carga modular: LoadAPI() y LoadWorker()
│   │
│   ├── database/
│   │   └── postgres.go          # Pool de conexiones SQL nativo y logger Warn
│   │
│   └── kafka/
│       └── consumer.go          # Consumer Group · worker pool · graceful drain
│
├── scripts/
│   └── entrypoint.sh            # Dispatcher de roles en contenedor (api | worker | monolith)
│
├── Dockerfile                   # Build multi-stage multi-binario
├── docker-compose.yml           # Orquestación completa (PostgreSQL, Kafka KRaft, API y Worker)
├── go.mod
└── go.sum
```

---

## Configuración

El servicio se configura **exclusivamente por variables de entorno**. No se requiere ningún archivo de configuración en producción.

### Variables para `magi-api` (Consultas)
| Variable | Obligatoria | Descripción | Valor por defecto |
|---|:---:|---|---|
| `DATABASE_DSN` | ✅ | Cadena de conexión PostgreSQL | — |
| `PORT` | ❌ | Puerto de escucha HTTP | `8080` |

### Variables para `magi-worker` (Ingestor)
| Variable | Obligatoria | Descripción | Valor por defecto |
|---|:---:|---|---|
| `DATABASE_DSN` | ✅ | Cadena de conexión PostgreSQL | — |
| `KAFKA_BROKERS` | ✅ | Lista de brokers de Kafka separados por comas | — |
| `KAFKA_STATIC_TOPIC` | ✅ | Tópico de mensajes AIS estáticos | `ais.static` |
| `KAFKA_DYNAMIC_TOPIC` | ✅ | Tópico de mensajes AIS dinámicos | `ais.dynamic` |
| `KAFKA_GROUP_ID` | ✅ | Consumer Group ID | `magi-consumer-group` |

> Para desarrollo local, crea un archivo `.env` en la raíz (ya excluido en `.gitignore`) y expórtalo antes de ejecutar el binario.

### Ejemplo `.env`

```env
DATABASE_DSN=host=localhost user=magi password=secret dbname=magi port=5432 sslmode=disable
PORT=8080
KAFKA_BROKERS=localhost:9092
KAFKA_STATIC_TOPIC=ais.static
KAFKA_DYNAMIC_TOPIC=ais.dynamic
KAFKA_GROUP_ID=magi-consumer-group
```

---

## Ejecución local

### Prerrequisitos

- Go 1.23+
- PostgreSQL corriendo y accesible
- Apache Kafka con los tópicos `ais.static` y `ais.dynamic` creados

```bash
# 1. Exportar variables de entorno (o usar un .env)
export $(cat .env | xargs)

# 2. Ejecutar la API de consultas (Terminal 1)
go run ./cmd/api/main.go

# 3. Ejecutar el Ingestor de Kafka (Terminal 2)
go run ./cmd/worker/main.go

# (Opcional) Ejecutar todo en un único proceso
go run ./cmd/main.go
```

---

## Docker y Docker Compose

### Orquestación completa con Docker Compose (Recomendado)

Levanta PostgreSQL 16, Apache Kafka (KRaft mode sin Zookeeper), el microservicio de consultas `magi-api` y el ingestor `magi-worker`:

```bash
docker compose up -d
```

Verificar el estado de los servicios:
```bash
docker compose ps
```

### Build y Ejecución Manual con Docker

```bash
# 1. Construir la imagen multi-binario
docker build -t magi/backend:latest .

# 2. Ejecutar la API HTTP (Read Path)
docker run -d \
  --name magi-api \
  -p 8080:8080 \
  -e DATABASE_DSN="host=db user=magi password=secret dbname=magi port=5432 sslmode=disable" \
  magi/backend:latest api

# 3. Ejecutar el Ingestor de Kafka (Write Path)
docker run -d \
  --name magi-worker \
  -e KAFKA_BROKERS=kafka:9092 \
  -e KAFKA_STATIC_TOPIC=ais.static \
  -e KAFKA_DYNAMIC_TOPIC=ais.dynamic \
  -e KAFKA_GROUP_ID=magi-consumer-group \
  -e DATABASE_DSN="host=db user=magi password=secret dbname=magi port=5432 sslmode=disable" \
  magi/backend:latest worker
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

### `GET /health` — Monitoreo y Health Check

Valida el estado del servicio y la conectividad real con PostgreSQL (`Ping`).

**Response** `200 OK`
```json
{
  "status": "healthy",
  "database": "connected"
}
```

**Response** `503 Service Unavailable` (Fallo de base de datos)
```json
{
  "status": "unhealthy",
  "error": "base de datos no responde al ping"
}
```

---

### `GET /ship/:imo` — Consultar buque por IMO

Devuelve la información del buque y su historial de posiciones paginado (orden cronológico inverso).

**Parámetros de ruta**

| Param | Tipo | Descripción |
|---|---|---|
| `imo` | `int` | Número IMO del buque (7 dígitos) |

**Parámetros de consulta (Query Params)**

| Param | Tipo | Default | Descripción |
|---|---|---|---|
| `limit` | `int` | `50` | Máximo de posiciones a retornar (Tope: 500) |
| `offset` | `int` | `0` | Desplazamiento para paginación |

**Response** `200 OK`
```json
{
  "imo": 9321483,
  "mmsi": 636092298,
  "callsign": "A8IG4",
  "shipname": "EVER GIVEN",
  "ship_type": "Container Ship",
  "total_positions_recorded": 1,
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
| `500` | Error interno de base de datos |

---

### `GET /ship/mmsi/:mmsi` — Consultar buque por MMSI

Devuelve la información del buque y su historial de telemetría paginado a partir de su identificador MMSI.

**Parámetros de ruta**

| Param | Tipo | Descripción |
|---|---|---|
| `mmsi` | `int` | Código MMSI del buque (9 dígitos) |

**Parámetros de consulta (Query Params):** `limit` (default: 50), `offset` (default: 0).

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
