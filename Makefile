include .env
export

export PROJECT_ROOT=$(shell pwd)

BACKEND_MAIN = cmd/api/main.go
BACKEND_BIN = ./bin/api
WORKER_MAIN = cmd/worker/main.go
WORKER_BIN = ./bin/worker

# app
dev:
	air

worker:
	go run $(WORKER_MAIN)

build: build-api build-worker

build-api: clean swagger
	go build -o $(BACKEND_BIN) $(BACKEND_MAIN)

build-worker:
	go build -o $(WORKER_BIN) $(WORKER_MAIN)

run-api:
	$(BACKEND_BIN)

run-worker:
	$(WORKER_BIN)

br: build-api run-api

br-worker: build-worker run-worker

clean: tidy fmt
	rm -rf ./bin

tidy:
	go mod tidy

fmt:
	go fmt ./...

swagger:
	swag init -g $(BACKEND_MAIN)

seed:
	go run cmd/seed/main.go

# docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-clean: docker-down
	rm -rf out/

docker-logs:
	docker compose logs -f

# migrations
migrate:
	docker compose run --rm postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"

migrate-up:
	@make migrate-action action=up

migrate-down:
	@make migrate-action action=down

# POSTGRES_HOST must be 'postgres' not 'localhost' when running api locally
migrate-action:
	docker compose run --rm postgres-migrate \
		-path /migrations \
		-database postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable \
		$(action)

# other
full-clean: clean docker-clean