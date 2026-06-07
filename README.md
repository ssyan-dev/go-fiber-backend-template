# Go Fiber Backend Template

⚡️ Production-ready REST API boilerplate built with Go and Fiber

## 🌟 Features

- **Clean Architecture:** Layers (Handler -> Service -> Repository)
- **Full Auth:** Support JWT (Access/Refresh) and OAuth2 (Google, Yandex, GitHub)

## 🛠 Stack

| Category | Technology |
|-----------|------------|
| **Programming language** | [Go 1.25+](go.dev) |
| **Framework** | [Fiber v3](docs.gofiber.io/) |
| **Database** | PostgreSQL [(pgx/v5)](github.com/jackc/pgx) |
| **Cache** | [redis](github.com/redis/go-redis) |
| **Logging** | [uber-go/zap](github.com/uber-go/zap) |
| **Documentation** | Swagger [(swaggo)](github.com/swaggo/swag) |
| **Configuration** | .env [(caarlos0/env)](github.com/caarlos0/env) |
## 📂 Project structure
```text
├── cmd/
│   ├── api/
│   ├── seed/
│   └── admin/
├── internal/
│   ├── auth/         # auth module
│   ├── user/         # users module
│   ├── middleware/   # middlewares
│   ├── models/       # sql models
│   └── pkg/          # utils
├── migrations/       # sql migrations
├── scripts/          # bash scripts
└── Makefile
```

## 🚀 Start

### 1. Automatic install
You can deploy everything in one click. The script will create the configuration, launch Docker, run migrations, and upload test data to the database.
```bash
chmod +x scripts/setup.sh && ./scripts/setup.sh
```

### 2. Running in development mode
To automatically restart the server when the code changes using **Air**:
```bash
make dev
```

### 3. Create administrator
You can create your first administrator manually:
```bash
./scripts/create-admin.sh admin@backend.com supercoolpassword
```

## 📝 API and Documentation

Swagger UI is available after starting the server at:
`http://localhost:8080/api/v1/docs/`

---

## 🛠 Makefile

### Development
- `make dev` — run with hot reload [(Air)](github.com/air-verse/air)
- `make build` — build app
- `make run` — run app
- `make swagger` — update swagger documentation
- `make clean` — clean builded app and format code
- `make fmt` — format code

### Postgres migrations
- `make migrate seq=<seq>` — create migration
- `make migrate-up` — up migration
- `make migrate-down` — down migration
- `make seed` — generate mock data

### Docker
- `make docker-up` — run containers
- `make docker-down` — stop containers
- `make docker-clean` — clear docker data
- `make docker-logs` — check docker logs

### Other
- `make full-clean` — clean all
