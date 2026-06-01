# URL Shortener

A production-grade URL shortener built in Go, deployed as a Docker cluster.

## Stages

### Stage 1: App + PostgreSQL

Go HTTP server with two endpoints backed by a Postgres database.

**Endpoints**

- `POST /shorten` accepts `{"url": "https://..."}`, generates a short code, inserts into Postgres, returns the code
- `GET /{code}` looks up the code and redirects to the original URL

**Short code generation** (`shortener.go`)

Base62 encoding over a random number in the range `[62^7, 62^8-1]` guarantees 7-character codes:

```go
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func encode(num uint64) string {
    var result string
    for num > 0 {
        index := num % 62
        result = string(charset[index]) + result
        num = num / 62
    }
    return result
}
```

**Database** (`init.sql`)

```sql
CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    original_url TEXT NOT NULL,
    short_code VARCHAR(10) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

`UNIQUE` on `short_code` means Postgres rejects duplicates at the database level.

**Connection** (`main.go`)

```go
db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
err = db.Ping()
```

`sql.Open` only validates the config. `db.Ping()` is what actually tests the connection.

---

### Stage 2: Redis Cache

Cache-aside pattern layered onto the redirect handler.

**Flow**

1. Check Redis for the code
2. Cache hit: redirect immediately, Postgres not touched
3. Cache miss: query Postgres, write result to Redis, then redirect

```go
cachedURL, err := rdb.Get(ctx, code).Result()
if err == nil {
    http.Redirect(w, r, cachedURL, http.StatusFound)
    return
}

// cache miss - query Postgres
err = db.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", code).Scan(&originalURL)

// store in Redis before redirecting
rdb.Set(ctx, code, originalURL, 0)
http.Redirect(w, r, originalURL, http.StatusFound)
```

`rdb.Set` TTL is `0` (no expiry). In production you would set a TTL like `24 * time.Hour`.

---

### Stage 3: Nginx Load Balancer

Three app instances running behind an Nginx reverse proxy.

**Scaling** (`docker-compose.yml`)

```yaml
app:
  build: .
  deploy:
    replicas: 3
```

**Nginx config** (`nginx.conf`)

```nginx
upstream app {
    server app:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://app;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

Docker's internal DNS resolves `app` to all 3 replicas. Nginx distributes requests round-robin by default. The `Server: nginx` header in responses confirms traffic is routed through the proxy.

All 3 instances share the same Postgres database and Redis cache, so a code shortened on instance 1 is resolvable on instance 2 or 3.

---

## Running

```
docker compose up --build
```

Requires a `.env` file:

```
POSTGRES_USER=
POSTGRES_PASSWORD=
POSTGRES_DB=
DATABASE_URL=postgresql://<user>:<pass>@db:5432/<dbname>?sslmode=disable
REDIS_URL=redis-cache:6379
```
