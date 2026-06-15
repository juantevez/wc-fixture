# wc-fixture

Sistema de gestión del fixture del Mundial 2026. Monorepo en Go con arquitectura de microservicios basada en DDD, CQRS y Event Sourcing.

## Arquitectura

El sistema está dividido en cinco bounded contexts independientes que se comunican de forma asíncrona a través de NATS JetStream. Cada contexto tiene su propia base de datos PostgreSQL.

```
                          ┌───────────────────┐
                          │  result-ingestion │
                          │    :8083          │
                          └────────┬──────────┘
                                   │ fixture.result.ingested
                                   ▼
┌──────────────┐    NATS    ┌──────────────────┐    fixture.*    ┌──────────────┐
│ team-registry│            │  fixture-core    │ ──────────────► │ notification │
│    :8082     │            │     :8080        │                 │    :8084     │
└──────────────┘            └──────────────────┘                 └──────────────┘
┌──────────────┐
│  venue-geo   │
│    :8081     │
└──────────────┘
```

### Patrones aplicados

- **DDD** — cada contexto tiene dominio, puertos y adaptadores separados
- **CQRS** — commands (escritura sobre el aggregate) y queries (lectura directa sobre read models)
- **Event Sourcing** — `fixture_events` es el event store append-only del aggregate `Fixture`
- **Hexagonal** — la lógica de dominio no depende de infraestructura; los adaptadores implementan puertos
- **Optimistic Locking** — campo `version` en el aggregate `Fixture` para evitar conflictos concurrentes

---

## Contextos

### fixture-core — `:8080`

Núcleo del sistema. Gestiona el torneo completo: fase de grupos, tabla de posiciones, bracket eliminatorio y resultados de partidos.

**Base de datos:** `fixture_db`

**API REST**

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/v1/tournaments/{id}/fixture` | Fixture completo (grupos + bracket) |
| GET | `/api/v1/tournaments/{id}/groups` | Listado de grupos con standings |
| GET | `/api/v1/tournaments/{id}/groups/{name}` | Detalle de un grupo |
| GET | `/api/v1/tournaments/{id}/groups/{name}/standings` | Tabla de posiciones |
| GET | `/api/v1/tournaments/{id}/best-thirds` | Ranking mejores terceros |
| GET | `/api/v1/tournaments/{id}/matches` | Partidos paginados con filtros |
| GET | `/api/v1/tournaments/{id}/matches/{matchID}` | Detalle de partido |
| POST | `/api/v1/tournaments/{id}/matches/{matchID}/result` | Registrar resultado |
| PUT | `/api/v1/tournaments/{id}/matches/{matchID}/schedule` | Actualizar horario |
| GET | `/api/v1/tournaments/{id}/knockout` | Bracket eliminatorio completo |
| GET | `/api/v1/tournaments/{id}/knockout/{phase}` | Ronda específica (`ROUND_OF_32`, `QUARTERFINAL`, `SEMIFINAL`, `THIRD_PLACE`, `FINAL`) |

**Eventos publicados (NATS)**

| Subject | Evento |
|---------|--------|
| `fixture.match.result_registered` | `MatchResultRegistered` |
| `fixture.group.stage_completed` | `GroupStageCompleted` |
| `fixture.knockout.bracket_generated` | `KnockoutBracketGenerated` |
| `fixture.knockout.match_advanced` | `KnockoutMatchAdvanced` |
| `fixture.match.schedule_updated` | `MatchScheduleUpdated` |
| `fixture.tournament.finished` | `TournamentFinished` |

**Evento consumido:** `fixture.result.ingested` → registra resultado en el aggregate

---

### venue-geo — `:8081`

Gestión de los 16 estadios sede con soporte geoespacial vía PostGIS. Incluye seed con las coordenadas oficiales FIFA de todos los venues (11 en USA, 2 en Canadá, 3 en México).

**Base de datos:** `venue_db`

**API REST**

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/v1/venues` | Listado de estadios (`?country=USA\|CAN\|MEX`) |
| GET | `/api/v1/venues/{venueID}` | Detalle de estadio |
| GET | `/api/v1/venues/distance` | Distancia geodésica entre dos venues (`?from=&to=`) |
| GET | `/api/v1/venues/distance-matrix` | Matriz de distancias entre todos los venues |
| GET | `/api/v1/venues/nearby` | Estadios dentro de un radio (`?lat=&lon=&radius_km=`) |

---

### team-registry — `:8082`

Registro de equipos nacionales y confederaciones FIFA. Incluye seed con las 6 confederaciones y sus cupos para el Mundial 2026 (48 equipos).

**Base de datos:** `team_db`

**API REST**

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/v1/teams` | Listado de equipos (filtros: `confederation`, `qualified`) |
| GET | `/api/v1/teams/{teamID}` | Detalle de equipo |
| GET | `/api/v1/confederations` | Las 6 confederaciones FIFA con cupos |
| GET | `/api/v1/confederations/{code}` | Detalle de confederación |

---

### result-ingestion — `:8083`

Ingesta de resultados desde sistemas externos hacia `fixture-core`. Garantiza idempotencia mediante una tabla de log con clave única por resultado. El endpoint está protegido con un token interno (`X-Internal-Token`).

**Base de datos:** `ingestion_db`

**API REST**

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/v1/results` | Ingestar resultado (requiere `X-Internal-Token`) |

Al recibir un resultado, publica en `fixture.result.ingested` para que `fixture-core` lo procese.

---

### notification — `:8084`

Consumer NATS puro. No expone API de negocio; solo sirve `/health`. Escucha eventos del stream `FIXTURE_EVENTS` y despacha notificaciones a los suscriptores del torneo.

**Eventos consumidos**

| Evento | Acción |
|--------|--------|
| `MatchResultRegistered` | Notifica resultado del partido |
| `KnockoutBracketGenerated` | Notifica generación del bracket |
| `TournamentFinished` | Notifica campeón del torneo |

---

## shared — Paquetes comunes

| Paquete | Contenido |
|---------|-----------|
| `pkg/events` | `DomainEvent` envelope, interfaz `Publisher`, subjects y stream NATS |
| `pkg/apperrors` | Errores de dominio tipados: `NotFound`, `Validation`, `Conflict`, `Unavailable` |
| `pkg/middleware` | `RequestID`, `Logging`, `Tracing` (OpenTelemetry), `Recover` |
| `pkg/logger` | Logger `slog` estructurado con propagación por contexto |
| `pkg/httputil` | Paginación (`PageParams`, `PageMeta`), helpers de encode/decode |
| `pkg/testutil` | Fixtures de prueba, helpers de assert, Testcontainers para tests de integración |

---

## Infraestructura

| Servicio | Imagen | Puerto |
|----------|--------|--------|
| PostgreSQL + PostGIS | `postgis/postgis:16-3.4` | `5432` |
| NATS JetStream | `nats:2.10-alpine` | `4222` (cliente) / `8222` (monitoring) |

**Bases de datos**

| Base | Contexto propietario |
|------|---------------------|
| `fixture_db` | fixture-core |
| `venue_db` | venue-geo |
| `team_db` | team-registry |
| `ingestion_db` | result-ingestion |

---

## Stack técnico

- **Go 1.25** con Go Workspaces (`go.work`)
- **chi v5** — router HTTP
- **pgx/v5** — driver PostgreSQL con connection pool
- **nats.go** — cliente NATS con JetStream (at-least-once, durable consumers, backoff progresivo)
- **PostGIS** — queries geoespaciales en venue-geo (`ST_DWithin`, `ST_Distance`, `GEOGRAPHY`)
- **OpenTelemetry** — tracing distribuido
- **Testcontainers** — tests de integración con contenedores reales

---

## Levantar el entorno local

```bash
cd deploy/docker

# Primera vez o tras modificar migraciones
docker compose --env-file .env down -v
docker compose --env-file .env up --build

# Arranque normal (volúmenes ya inicializados)
docker compose --env-file .env up -d
```

> Las migraciones SQL se aplican automáticamente en el primer arranque de PostgreSQL via `init-db.sh`. Si se modifican archivos en `contexts/*/migrations/`, hay que hacer `down -v` para que se re-ejecuten sobre un volumen limpio.

**Variables de entorno** — ver `deploy/docker/.env`:

| Variable | Descripción |
|----------|-------------|
| `POSTGRES_USER` / `POSTGRES_PASSWORD` | Credenciales de PostgreSQL |
| `FIXTURE_DB_URL` / `VENUE_DB_URL` / `TEAM_DB_URL` / `INGESTION_DB_URL` | DSN por contexto |
| `NATS_URL` | URL de conexión a NATS |
| `INTERNAL_TOKEN` | Token compartido entre `result-ingestion` y `fixture-core` |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` |
| `ENV` | `development` / `production` |

**Verificar estado**

```bash
docker ps --format "table {{.Names}}\t{{.Status}}"
```

---

## Estructura del monorepo

```
wc-fixture/
├── contexts/
│   ├── fixture-core/
│   │   ├── cmd/server/          # main.go + wire.go (DI manual)
│   │   ├── internal/
│   │   │   ├── domain/          # aggregate Fixture, value objects, ports
│   │   │   ├── application/     # commands/ y queries/ (CQRS)
│   │   │   └── infrastructure/  # postgres/, nats/, http/
│   │   └── migrations/          # 000001_…000007_ SQL
│   ├── venue-geo/
│   ├── team-registry/
│   ├── result-ingestion/
│   └── notification/
├── shared/
│   └── pkg/                     # events, apperrors, middleware, logger, httputil, testutil
├── deploy/
│   ├── docker/                  # docker-compose.yml, docker-compose.override.yml, .env
│   ├── postgres/                # init-db.sh (crea bases, extensiones y corre migraciones)
│   └── nats/                    # nats.conf
└── go.work
```
