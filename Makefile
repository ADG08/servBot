# ServBot — Docker & run
# Prod avec registry: make prod-build VERSION=1.0.0 && make push REGISTRY=ghcr.io/user/servbot VERSION=1.0.0
.PHONY: build up down logs prod-up prod-down prod-logs shell clean push \
        migrate-up migrate-down migrate-create migrate-force migrate-version

# Dev (compose simple)
build:
	docker compose build

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

# Prod (compose + override prod)
prod-build:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml build

prod-up:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

prod-down:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml down

prod-logs:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml logs -f

# Utilitaires
shell:
	docker compose run --rm --entrypoint sh bot

clean:
	docker compose down -v
	docker rmi servbot:latest 2>/dev/null || true

# Build image pour prod puis push vers un registry (ex: ghcr.io/user/servbot)
VERSION ?= latest
REGISTRY ?=
push: prod-build
	@test -n "$(REGISTRY)" || (echo "REGISTRY=... requis (ex: ghcr.io/user/servbot)" && exit 1)
	docker tag servbot:latest $(REGISTRY):$(VERSION)
	docker push $(REGISTRY):$(VERSION)

# ─── Migrations (golang-migrate) ─────────────────────────────────────────────
# Les migrations s'appliquent automatiquement au démarrage du bot.
# Ces targets permettent un contrôle manuel (dev, debug, rollback).
# L'URL de la base est définie dans le service "migrate" de docker-compose.yml.

migrate-up:
	docker compose run --rm migrate up

migrate-down:
	docker compose run --rm migrate down 1

migrate-force:
	@test -n "$(V)" || (echo "V=... requis (ex: make migrate-force V=2)" && exit 1)
	docker compose run --rm migrate force $(V)

migrate-version:
	docker compose run --rm migrate version

migrate-create:
	@test -n "$(NAME)" || (echo "NAME=... requis (ex: make migrate-create NAME=add_column_foo)" && exit 1)
	docker compose run --rm --no-deps --entrypoint migrate migrate create -ext sql -dir /migrations -seq -digits 6 $(NAME)
