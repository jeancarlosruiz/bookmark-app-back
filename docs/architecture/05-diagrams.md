---
title: Architecture Diagrams
created: 2025-01-15
updated: 2026-03-08
---

# Diagramas de Arquitectura: Go-Bookmark API

> Estos diagramas complementan el documento [04-system-design.md](./04-system-design.md)
> Formato: Mermaid (compatible con GitHub, Notion, Obsidian, etc.)

---

## Tabla de Contenidos

1. [Request Lifecycle](#1-request-lifecycle)
2. [Authentication Flow (JWKS)](#2-authentication-flow-jwks)
3. [Cache Flow (Hit vs Miss)](#3-cache-flow-hit-vs-miss)
4. [Entity Relationship Diagram](#4-entity-relationship-diagram-erd)
5. [Component Diagram](#5-component-diagram)
6. [Deployment Architecture](#6-deployment-architecture)

---

## 1. Request Lifecycle

Flujo completo desde que llega un request HTTP hasta la respuesta.

```mermaid
flowchart TB
    subgraph Client
        A[HTTP Request]
    end

    subgraph "Gin Router"
        B[CORS Middleware]
        C[Protect Middleware]
        D[Validator Middleware]
    end

    subgraph "Application Layer"
        E[Controller]
        F[Service]
        G[Repository]
    end

    subgraph "Data Layer"
        H[(PostgreSQL)]
        I[(Redis Cache)]
    end

    A --> B
    B --> C
    C -->|Valid JWT| D
    C -->|Invalid JWT| X[401 Unauthorized]
    D -->|Valid Payload| E
    D -->|Invalid Payload| Y[422 Validation Error]

    E --> F
    F -->|Check Cache| I
    I -->|Cache Hit| F
    I -->|Cache Miss| G
    G --> H
    H --> G
    G -->|Populate Cache| I
    G --> F
    F --> E
    E --> Z[JSON Response]

    style X fill:#ff6b6b
    style Y fill:#ffa94d
    style Z fill:#69db7c
```

---

## 2. Authentication Flow (JWKS)

Como funciona la autenticacion con JWKS y JWT EdDSA.

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant API as Go-Bookmark API
    participant JWKS as Better Auth<br/>(JWKS Endpoint)
    participant Cache as In-Memory<br/>Key Cache

    Note over API,Cache: Startup: Fetch JWKS keys
    API->>JWKS: GET /api/auth/jwks
    JWKS-->>API: Public Keys (EdDSA)
    API->>Cache: Store keys (refresh: 1hr)

    Note over Client,API: Request with JWT
    Client->>API: GET /api/bookmarks<br/>Authorization: Bearer <token>

    API->>API: Extract token from header
    API->>Cache: Get public key
    Cache-->>API: EdDSA public key

    API->>API: Verify signature (EdDSA)
    API->>API: Check expiration
    API->>API: Extract claims:<br/>user_id, email, name

    alt Token Valid
        API->>API: Set context:<br/>c.Set("user_id", ...)
        API-->>Client: 200 OK + Data
    else Token Invalid/Expired
        API-->>Client: 401 Unauthorized
    end

    Note over API,Cache: Background: Key Refresh
    loop Every 1 hour
        API->>JWKS: GET /api/auth/jwks
        JWKS-->>API: Updated keys
        API->>Cache: Refresh keys
    end
```

---

## 3. Cache Flow (Hit vs Miss)

Patron Cache-Aside con tiempos de respuesta.

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Service as Bookmark<br/>Service
    participant Cache as Redis<br/>Cache
    participant DB as PostgreSQL

    Note over Client,DB: Scenario 1: Cache HIT (~5ms)
    Client->>Service: GetBookmarks(userID)
    Service->>Cache: GET bookmarks:user:{id}
    Cache-->>Service: Cached data found
    Service-->>Client: Return bookmarks
    Note right of Client: Response: ~5ms

    Note over Client,DB: Scenario 2: Cache MISS (~150ms)
    Client->>Service: GetBookmarks(userID)
    Service->>Cache: GET bookmarks:user:{id}
    Cache-->>Service: nil (not found)
    Service->>DB: SELECT * FROM bookmarks<br/>WHERE user_id = ?
    DB-->>Service: Bookmark records
    Service->>Cache: SET bookmarks:user:{id}<br/>TTL: 10 minutes
    Cache-->>Service: OK
    Service-->>Client: Return bookmarks
    Note right of Client: Response: ~150ms

    Note over Client,DB: Scenario 3: Write Operation (Invalidation)
    Client->>Service: CreateBookmark(data)
    Service->>DB: INSERT INTO bookmarks
    DB-->>Service: Created
    Service->>Cache: DEL bookmarks:user:{id}
    Service->>Cache: DEL bookmarks:user:{id}:archived
    Service->>Cache: DEL tags:user:{id}
    Note right of Cache: All user caches<br/>invalidated
    Service-->>Client: Return new bookmark

    Note over Client,DB: Scenario 4: Redis Unavailable (Graceful Degradation)
    Client->>Service: GetBookmarks(userID)
    Service->>Cache: PING
    Cache-->>Service: Connection error
    Service->>DB: SELECT * FROM bookmarks<br/>WHERE user_id = ?
    DB-->>Service: Bookmark records
    Service-->>Client: Return bookmarks
    Note right of Client: App continues working<br/>without cache
```

---

## 4. Entity Relationship Diagram (ERD)

Modelo de datos con relaciones y constraints.

```mermaid
erDiagram
    USERS_SYNC {
        uuid id PK
        string email UK
        string name
        timestamp created_at
        timestamp updated_at
    }

    BOOKMARKS {
        uuid id PK
        uuid user_id FK
        string title
        string url
        string normalized_url UK
        string description
        string favicon
        boolean is_archived
        boolean is_pinned
        int visit_count
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    TAGS {
        uuid id PK
        uuid user_id FK
        string name
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    BOOKMARK_TAGS {
        uuid bookmark_id FK
        uuid tag_id FK
    }

    USERS_SYNC ||--o{ BOOKMARKS : "has many"
    USERS_SYNC ||--o{ TAGS : "has many"
    BOOKMARKS }o--o{ TAGS : "many to many"
    BOOKMARKS ||--o{ BOOKMARK_TAGS : "junction"
    TAGS ||--o{ BOOKMARK_TAGS : "junction"
```

### Notas del ERD

| Tabla | Schema | Notas |
|-------|--------|-------|
| `users_sync` | `neon_auth` | Externa, gestionada por Better Auth |
| `bookmarks` | `public` | Soft delete via `deleted_at` |
| `tags` | `public` | User-scoped (cada usuario tiene sus propios tags) |
| `bookmark_tags` | `public` | Join table para N:M |

### Indices

```sql
-- Evita duplicados de URL por usuario
CREATE UNIQUE INDEX idx_user_url ON bookmarks(user_id, normalized_url);

-- Busqueda rapida por usuario
CREATE INDEX idx_bookmarks_user_id ON bookmarks(user_id);
CREATE INDEX idx_tags_user_id ON tags(user_id);

-- Ordenamiento por fecha
CREATE INDEX idx_bookmarks_created_at ON bookmarks(created_at);
```

---

## 5. Component Diagram

Estructura de componentes internos y sus dependencias.

```mermaid
flowchart TB
    subgraph "Entry Point"
        MAIN[main.go]
    end

    subgraph "HTTP Layer"
        ROUTER[Gin Router]
        subgraph "Middlewares"
            MW_CORS[CORS]
            MW_AUTH[Protect<br/>JWT/JWKS]
            MW_VAL[Validator<br/>Generic T]
        end
    end

    subgraph "Controllers"
        CTRL_BM[BookmarkController]
        CTRL_TAG[TagController]
    end

    subgraph "Services"
        SVC_BM[BookmarkService]
        SVC_TAG[TagService]
        SVC_CACHE[CacheService]
        SVC_SCRAPE[ScrapperService]
    end

    subgraph "Repositories"
        REPO_BM[BookmarkRepository]
        REPO_TAG[TagRepository]
    end

    subgraph "Infrastructure"
        DB_PG[(PostgreSQL)]
        DB_REDIS[(Redis)]
        EXT_JWKS[JWKS Endpoint]
        EXT_WEB[External URLs<br/>Metadata]
    end

    subgraph "Shared"
        CONFIG[Config]
        MODELS[Models]
        UTILS[Utils<br/>URL Normalize]
        VALIDATOR[Validator<br/>Schemas]
    end

    MAIN --> ROUTER
    MAIN --> CONFIG
    MAIN --> DB_PG
    MAIN --> DB_REDIS

    ROUTER --> MW_CORS
    MW_CORS --> MW_AUTH
    MW_AUTH --> MW_VAL
    MW_AUTH -.-> EXT_JWKS
    MW_VAL --> CTRL_BM
    MW_VAL --> CTRL_TAG
    MW_VAL -.-> VALIDATOR

    CTRL_BM --> SVC_BM
    CTRL_TAG --> SVC_TAG

    SVC_BM --> REPO_BM
    SVC_BM --> SVC_TAG
    SVC_BM --> SVC_CACHE
    SVC_BM --> SVC_SCRAPE
    SVC_BM -.-> UTILS

    SVC_TAG --> REPO_TAG
    SVC_TAG --> SVC_CACHE

    SVC_CACHE --> DB_REDIS
    SVC_SCRAPE -.-> EXT_WEB

    REPO_BM --> DB_PG
    REPO_TAG --> DB_PG
    REPO_BM -.-> MODELS
    REPO_TAG -.-> MODELS

    style DB_PG fill:#336791
    style DB_REDIS fill:#dc382d
    style EXT_JWKS fill:#f0ad4e
    style EXT_WEB fill:#5bc0de
```

### Leyenda

| Color | Significado |
|-------|-------------|
| Azul | PostgreSQL (persistencia) |
| Rojo | Redis (cache) |
| Amarillo | Servicio externo autenticacion |
| Celeste | Servicios externos web |
| Linea punteada | Dependencia opcional/config |

---

## 6. Deployment Architecture

Arquitectura de despliegue en produccion.

```mermaid
flowchart TB
    subgraph "Internet"
        USER[Users]
    end

    subgraph "CDN / Edge"
        CF[Cloudflare<br/>SSL Termination]
    end

    subgraph "Platform (Railway/Vercel)"
        LB[Load Balancer]

        subgraph "API Instances"
            API1[Go API #1]
            API2[Go API #2]
            API3[Go API #3]
        end
    end

    subgraph "Auth Provider"
        AUTH[Better Auth<br/>JWKS Endpoint]
    end

    subgraph "Data Layer (Managed)"
        subgraph "Neon PostgreSQL"
            PG_PRIMARY[(Primary)]
            PG_REPLICA[(Read Replica)]
        end

        subgraph "Upstash Redis"
            REDIS_PRIMARY[(Primary)]
            REDIS_REPLICA[(Replica)]
        end
    end

    USER -->|HTTPS| CF
    CF -->|HTTP| LB
    LB --> API1
    LB --> API2
    LB --> API3

    API1 & API2 & API3 -->|Verify JWT| AUTH
    API1 & API2 & API3 -->|Read/Write| PG_PRIMARY
    API1 & API2 & API3 -->|Read| PG_REPLICA
    API1 & API2 & API3 -->|Cache| REDIS_PRIMARY

    REDIS_PRIMARY -.->|Sync| REDIS_REPLICA
    PG_PRIMARY -.->|Sync| PG_REPLICA

    style CF fill:#f38020
    style LB fill:#764abc
    style AUTH fill:#00d4aa
    style PG_PRIMARY fill:#336791
    style PG_REPLICA fill:#336791
    style REDIS_PRIMARY fill:#dc382d
    style REDIS_REPLICA fill:#dc382d
```

### Flujo de Datos en Produccion

```mermaid
sequenceDiagram
    participant User
    participant CDN as Cloudflare
    participant LB as Load Balancer
    participant API as Go API Instance
    participant Auth as Better Auth
    participant Cache as Redis
    participant DB as PostgreSQL

    User->>CDN: HTTPS Request
    Note right of CDN: SSL Termination
    CDN->>LB: HTTP Request
    Note right of LB: Round Robin
    LB->>API: Forward to instance

    API->>Auth: Verify JWT (cached keys)
    Auth-->>API: Valid

    API->>Cache: Check cache
    alt Cache Hit
        Cache-->>API: Return data
    else Cache Miss
        API->>DB: Query
        DB-->>API: Data
        API->>Cache: Populate
    end

    API-->>LB: Response
    LB-->>CDN: Response
    CDN-->>User: HTTPS Response
```

### Caracteristicas de Produccion

| Componente | Servicio | Caracteristica |
|------------|----------|----------------|
| **SSL** | Cloudflare | Termination en edge, certificados automaticos |
| **Load Balancer** | Railway | Round Robin, health checks |
| **API** | Railway | Auto-scaling, zero-downtime deploys |
| **Auth** | Better Auth | JWKS con rotacion automatica |
| **Database** | Neon | Connection pooling, read replicas |
| **Cache** | Upstash | Serverless Redis, global replication |

---

## Como Usar Estos Diagramas

### En GitHub

Los archivos `.md` con bloques de codigo Mermaid se renderizan automaticamente.

### En Notion

1. Crear bloque de codigo
2. Seleccionar lenguaje "Mermaid"
3. Pegar el codigo

### En Presentaciones

Usar [Mermaid Live Editor](https://mermaid.live/) para exportar como:
- PNG (alta resolucion)
- SVG (vectorial)

### En Portfolio Web

```jsx
// Con react-mermaid o similar
import Mermaid from 'react-mermaid';

<Mermaid chart={`
  flowchart TB
    A --> B
`} />
```

---

## Herramientas Alternativas

Si prefieres diagramas mas visuales:

| Herramienta | Uso | Link |
|-------------|-----|------|
| **Excalidraw** | Diagramas hand-drawn style | [excalidraw.com](https://excalidraw.com) |
| **draw.io** | Diagramas profesionales | [draw.io](https://draw.io) |
| **Figma** | Diseno personalizado | [figma.com](https://figma.com) |
| **Lucidchart** | Diagramas colaborativos | [lucidchart.com](https://lucidchart.com) |

---

> Estos diagramas complementan la documentacion tecnica y son ideales para presentaciones de portfolio o entrevistas tecnicas.
