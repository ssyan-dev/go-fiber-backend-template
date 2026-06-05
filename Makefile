include .env
export

export PROJECT_ROOT=$(shell pwd)

BACKEND_MAIN = cmd/api/main.go
BACKEND_BIN = ./bin/api

# app
dev:
	air

build: clean swagger
	go build -o $(BACKEND_BIN) $(BACKEND_MAIN)

run:
	$(BACKEND_BIN)

br: build run

clean: tidy fmt
	rm -rf ./bin

tidy:
	go mod tidy

fmt:
	go fmt ./...

swagger:
	swag init -g $(BACKEND_MAIN)

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