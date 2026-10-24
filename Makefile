COMPOSE ?= docker compose
COMPOSE_FILE ?= docker-compose.yml
COMPOSE_LOCAL ?= docker-compose.local.yml
IMAGE ?= erriour/databin
VERSION := $(shell sed 's/^[[:space:]]*//;s/[[:space:]]*$$//' version.txt)

.PHONY: help env deploy up down logs pull restart local local-down redis dev publish

help:
	@echo "Production:"
	@echo "  make deploy   pull image and start (docker-compose.yml)"
	@echo "  make down     stop stack"
	@echo "  make logs     follow app logs"
	@echo "  make restart  down + deploy"
	@echo ""
	@echo "Local Docker build:"
	@echo "  make local    build and start (docker-compose.local.yml)"
	@echo "  make local-down"
	@echo ""
	@echo "Go on host:"
	@echo "  make redis    redis only"
	@echo "  make dev      run app"
	@echo ""
	@echo "Maintainer:"
	@echo "  make publish  build and push $(IMAGE) (version from version.txt)"

env:
	@test -f .env || cp .env.example .env

pull:
	$(COMPOSE) -f $(COMPOSE_FILE) pull app

deploy: env pull
	$(COMPOSE) -f $(COMPOSE_FILE) up -d

up: deploy

down:
	$(COMPOSE) -f $(COMPOSE_FILE) down

logs:
	$(COMPOSE) -f $(COMPOSE_FILE) logs -f app

restart: down deploy

local: env
	$(COMPOSE) -f $(COMPOSE_LOCAL) up --build -d

local-down:
	$(COMPOSE) -f $(COMPOSE_LOCAL) down

redis: env
	$(COMPOSE) -f $(COMPOSE_FILE) up redis -d

dev:
	go run .

publish:
	docker build -t $(IMAGE):latest -t $(IMAGE):$(VERSION) .
	docker push $(IMAGE):latest
	docker push $(IMAGE):$(VERSION)
