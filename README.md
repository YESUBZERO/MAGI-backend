<div align="center">

# AIRES · MAGI (Monorepo)

**Plataforma Fullstack de Monitoreo Marítimo, Telemetría AIS y Análisis en Tiempo Real**

*Sistema [MAGI — Monitoreo y Análisis de Gases e Inmisiones]*

![Next.js](https://img.shields.io/badge/Next.js-16+-black?style=flat-square&logo=next.dot.js&logoColor=white)
![Turborepo](https://img.shields.io/badge/Turborepo-Monorepo-EF4444?style=flat-square&logo=turborepo&logoColor=white)
![Bun](https://img.shields.io/badge/Bun-1.3+-fbf0df?style=flat-square&logo=bun&logoColor=black)
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)
![Kafka](https://img.shields.io/badge/Kafka-kafka--go-231F20?style=flat-square&logo=apachekafka&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white)
![Zitadel](https://img.shields.io/badge/Zitadel-OIDC_Auth-3B82F6?style=flat-square&logo=auth0&logoColor=white)

</div>

---

## 🏗️ Arquitectura del Monorepositorio

El proyecto está organizado como un monorepositorio optimizado con **Turborepo** y gestionado por **Bun**:

```
aires-magi/
├── apps/
│   ├── web/                     # Frontend Next.js App Router (TypeScript, Tailwind, shadcn-ui)
│   │   ├── src/app/             # Páginas, layout y NextAuth dynamic route (/api/auth/[...nextauth])
│   │   ├── src/components/      # Componentes UI (AISDashboard, TurnstileWidget, Header, ModeToggle)
│   │   ├── src/lib/             # Configuración de Zitadel OIDC Provider (NextAuth)
│   │   └── src/stores/          # Store de Zustand para telemetría marítima y filtros
│   │
│   └── server/                  # Backend Go con patrón CQRS
│       ├── api/                 # Especificación de OpenAPI 3.0 (openapi.yaml)
│       ├── cmd/
│       │   ├── server/          # Microservicio HTTP de Consultas (Read Path + Gin)
│       │   ├── worker/          # Microservicio Ingestor de Kafka (Write Path + kafka-go)
│       │   ├── all/             # Binario unificado para desarrollo local
│       │   └── openapi/         # Servidor de especificación OpenAPI
│       └── internal/
│           ├── ais/             # Repositorio GORM (idempotencia) y Servicios CQRS
│           ├── cache/           # Cliente Redis (go-redis/v9)
│           ├── config/          # Carga de configuración jerárquica con Viper
│           ├── database/        # Conexión optimizada a PostgreSQL con pool
│           ├── handlers/        # Handlers HTTP de Gin con Zerolog y Healthcheck
│           ├── messaging/       # Consumer Group concurrente y Producer de Kafka (kafka-go)
│           ├── models/          # Entidades de dominio y tablas GORM
│           └── observability/   # Instrumentación OpenTelemetry distribuida
│
├── packages/
│   ├── config/                  # Configuraciones base compartidas de TypeScript
│   ├── db/                      # Documentación y esquemas de base de datos
│   └── env/                     # Validación de variables de entorno seguras con Zod (@t3-oss/env-nextjs)
│
├── scripts/
│   ├── native/                  # Scripts de desarrollo de Turborepo
│   └── entrypoint.sh            # Dispatcher CQRS para contenedores Docker
│
├── docker-compose.yml           # Orquestación completa (Postgres, Kafka, Redis, API, Worker, Web)
├── package.json                 # Workspaces y scripts globales
└── turbo.json                   # Pipeline de tareas y caché inteligente de Turborepo
```

---

## ⚡ Stack Tecnológico

| Capa | Tecnologías |
|---|---|
| **Gestor de Monorepo** | Turborepo + Bun Workspaces |
| **Frontend (`apps/web`)** | Next.js 16 (App Router), React 19, TypeScript, Tailwind CSS v4, shadcn-ui (zinc/vega), Zustand, React Hook Form |
| **Autenticación Frontend** | Zitadel OIDC / OAuth 2.0 vía NextAuth.js con sesión JWT |
| **Protección contra Bots** | Cloudflare Turnstile (`@marsidev/react-turnstile`) |
| **Backend API (`apps/server`)** | Go 1.25, Gin Gonic, Zerolog, Viper, OpenTelemetry, oapi-codegen |
| **Ingesta de Eventos** | Apache Kafka (KRaft mode) + `segmentio/kafka-go` (Consumer Group con Worker Pool) |
| **Persistencia** | PostgreSQL 16 + GORM (con OnConflict DoNothing para idempotencia en inserciones) |
| **Caché en Memoria** | Redis 7 + `go-redis/v9` |
| **Pruebas y Calidad** | Testify (`apps/server`), Vitest (`apps/web`), golangci-lint |

---

## 🚀 Inicio Rápido

### Prerrequisitos

- [Bun](https://bun.sh/) (v1.2+)
- [Go](https://go.dev/) (v1.23+)
- [Docker & Docker Compose](https://www.docker.com/)

### 1. Instalación de Dependencias

```bash
# Instala dependencias del frontend y monorepo
bun install

# Sincroniza dependencias del backend Go
cd apps/server && go mod tidy && cd ../..
```

### 2. Variables de Entorno

Configura las variables de entorno en el frontend y backend:

```bash
# Frontend
cp apps/web/.env.example apps/web/.env.local

# Backend
cp apps/server/.env.example apps/server/.env
```

#### Parámetros clave de Zitadel en `apps/web/.env.local`:
```env
NEXTAUTH_URL=http://localhost:3001
NEXTAUTH_SECRET=genera-una-cadena-secreta-aleatoria-de-32-caracteres
ZITADEL_ISSUER=https://<tu-instancia>.zitadel.cloud
ZITADEL_CLIENT_ID=<tu-client-id>
ZITADEL_CLIENT_SECRET=<tu-client-secret>
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_TURNSTILE_SITE_KEY=1x00000000000000000000AA
```

### 3. Ejecutar en Desarrollo

```bash
# Iniciar Frontend (Next.js) y Backend API (Go) concurrentemente:
bun run dev

# O iniciar la suite completa (Frontend + Backend Unificado con Kafka Worker):
bun run dev:all
```

- **Frontend:** [http://localhost:3001](http://localhost:3001)
- **Backend API:** [http://localhost:8080](http://localhost:8080)
- **API Health:** [http://localhost:8080/health](http://localhost:8080/health)

---

## 🧪 Pruebas y Validación

```bash
# Ejecutar todas las pruebas (Frontend + Backend)
bun run test

# Ejecutar pruebas unitarias de Go (con Testify)
bun run test:server

# Validar tipos TypeScript en todo el monorepo con Turborepo
bun run check-types

# Construir producción
bun run build
```

---

## 🐳 Despliegue con Docker Compose

Para levantar la infraestructura completa con todos los microservicios:

```bash
docker compose up -d --build
```

Servicios levantados:
1. **`magi-web`**: Aplicación Next.js lista para producción (puerto `3001`).
2. **`magi-api`**: Microservicio HTTP de consultas de buques (puerto `8080`).
3. **`magi-worker`**: Microservicio ingestor de Kafka con worker pool concurrente.
4. **`postgres`**: Base de datos PostgreSQL 16 con volumen persistente.
5. **`kafka`**: Broker Apache Kafka en modo KRaft (puerto `9092`).
6. **`redis`**: Instancia de caché Redis 7 (puerto `6379`).

---

## 📖 Endpoints Principales de la API (OpenAPI 3.0)

| Método | Endpoint | Descripción |
|---|---|---|
| `GET` | `/health` | Healthcheck profundo (verifica conexión a PostgreSQL) |
| `POST` | `/api/v1/static` | Registro/actualización de identidad de buque (MMSI, IMO, Shipname) |
| `GET` | `/api/v1/ship/:imo` | Consulta de buque por IMO con telemetría dinámica paginada (`?limit=50&offset=0`) |
| `GET` | `/api/v1/ship/mmsi/:mmsi` | Consulta de buque por MMSI con telemetría dinámica paginada (`?limit=50&offset=0`) |
