#!/bin/bash

set -e

echo "starting setup..."

if [ ! -f .env ]; then
    echo "creating .env from .env.example.."
    cp .env.example .env
else
    echo ".env file already exists."
fi

echo "starting docker..."
make docker-up

echo "waiting for postgres to be ready..."
until docker compose exec postgres pg_isready -U postgres > /dev/null 2>&1; do
  echo "still waiting for postgres..."
  sleep 2
done

echo "running database migrations..."
make migrate-up

echo "seeding database..."
make seed

echo "setup complete! 'make dev' to start app in dev mode"
