# ServBot — Docker & run
# Prod avec registry: make prod-build VERSION=1.0.0 && make push REGISTRY=ghcr.io/user/servbot VERSION=1.0.0
.PHONY: build up down logs prod-up prod-down prod-logs shell clean push

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
