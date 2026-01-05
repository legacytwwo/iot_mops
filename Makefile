SHELL := /bin/bash

ROOT := $(abspath .)
INFRA_DIR := gitops
RULE_DIR := rule_engine/deploy
IOT_CONTROLLER_DIR := iot_controller/deploy
SIMULATOR_DIR := data_simulator/deploy

COMPOSE_INFRA := docker compose -f $(INFRA_DIR)/compose.yml --project-directory $(INFRA_DIR)
COMPOSE_RULE := docker compose -f $(RULE_DIR)/compose.yml --project-directory $(RULE_DIR)
COMPOSE_CONTROLLER := docker compose -f $(IOT_CONTROLLER_DIR)/compose.yml --project-directory $(IOT_CONTROLLER_DIR)
COMPOSE_SIMULATOR := docker compose -f $(SIMULATOR_DIR)/compose.yml --project-directory $(SIMULATOR_DIR)

ifneq ("$(wildcard gitops/.env)","")
include gitops/.env
export
endif

MONGO_USERNAME ?= root
MONGO_PASSWORD ?= example
MONGO_URI ?= mongodb://$(MONGO_USERNAME):$(MONGO_PASSWORD)@localhost:27017/?authSource=admin
MONGO_DB ?= iot
MONGO_CONNECT_TIMEOUT ?= 5s

.PHONY: help \
        infra-up infra-up-prod infra-down infra-restart infra-logs infra-ps \
        local-up local-down local-build local-rebuild local-logs local-ps \
        rule-up rule-down rule-restart rule-build rule-rebuild rule-logs rule-ps \
        controller-up controller-down controller-restart controller-build controller-rebuild controller-logs controller-ps \
        simulator-up simulator-down simulator-restart simulator-build simulator-rebuild simulator-logs simulator-ps \
        deploy-rule-up deploy-rule-down deploy-controller-up deploy-controller-down deploy-simulator-up deploy-simulator-down \
        migrate test smoke-rule

###############################################################################
# • help center ##
###############################################################################
help: ## Show available targets
	@echo ""; \
	echo "$$(printf '\033[1m• Available targets\033[0m')"; \
	grep -E '^[a-zA-Z0-9_-]+:.*?##' $(MAKEFILE_LIST) | \
	awk 'BEGIN{FS=":.*?## "}; {printf "  \033[36m%-24s\033[0m %s\n", $$1,$$2}'; \
	echo

###############################################################################
# • infra (gitops) ##
###############################################################################
infra-up: ## start infra (gitops/compose.yml)
	$(COMPOSE_INFRA) up -d

infra-up-prod: ## start infra with watchtower profile
	$(COMPOSE_INFRA) --profile prod up -d

infra-down: ## stop infra
	$(COMPOSE_INFRA) down --remove-orphans

infra-restart: ## restart infra
	$(COMPOSE_INFRA) restart

infra-logs: ## tail infra logs
	$(COMPOSE_INFRA) logs -f

infra-ps: ## show infra status
	$(COMPOSE_INFRA) ps

###############################################################################
# • local all-in-one ##
###############################################################################
local-up: ## start infra + all local services
	$(MAKE) infra-up
	$(MAKE) rule-up
	$(MAKE) controller-up
	$(MAKE) simulator-up

local-down: ## stop all local services + infra
	$(MAKE) simulator-down
	$(MAKE) controller-down
	$(MAKE) rule-down
	$(MAKE) infra-down

local-build: ## build all local service images
	$(MAKE) rule-build
	$(MAKE) controller-build
	$(MAKE) simulator-build

local-rebuild: ## rebuild and restart all local services
	$(MAKE) rule-rebuild
	$(MAKE) controller-rebuild
	$(MAKE) simulator-rebuild

local-logs: ## show log commands for local stacks
	@echo "Use:"
	@echo "  make rule-logs"
	@echo "  make controller-logs"
	@echo "  make simulator-logs"

local-ps: ## show status for all local stacks
	$(COMPOSE_RULE) ps
	$(COMPOSE_CONTROLLER) ps
	$(COMPOSE_SIMULATOR) ps

###############################################################################
# • rule_engine (local) ##
###############################################################################
rule-up: ## start rule_engine stack (rule_engine/deploy/compose.yml)
	$(COMPOSE_RULE) up -d

rule-down: ## stop rule_engine stack
	$(COMPOSE_RULE) down --remove-orphans

rule-restart: ## restart rule_engine stack
	$(COMPOSE_RULE) restart

rule-build: ## build rule_engine image
	$(COMPOSE_RULE) build rule_engine

rule-rebuild: ## rebuild and restart rule_engine stack
	$(COMPOSE_RULE) build rule_engine
	$(COMPOSE_RULE) up -d

rule-logs: ## tail rule_engine logs
	$(COMPOSE_RULE) logs -f

rule-ps: ## show rule_engine status
	$(COMPOSE_RULE) ps

###############################################################################
# • iot_controller (local) ##
###############################################################################
controller-up: ## start iot_controller stack
	$(COMPOSE_CONTROLLER) up -d

controller-down: ## stop iot_controller stack
	$(COMPOSE_CONTROLLER) down --remove-orphans

controller-restart: ## restart iot_controller stack
	$(COMPOSE_CONTROLLER) restart

controller-build: ## build iot_controller image
	$(COMPOSE_CONTROLLER) build iot_controller

controller-rebuild: ## rebuild and restart iot_controller
	$(COMPOSE_CONTROLLER) build iot_controller
	$(COMPOSE_CONTROLLER) up -d

controller-logs: ## tail iot_controller logs
	$(COMPOSE_CONTROLLER) logs -f

controller-ps: ## show iot_controller status
	$(COMPOSE_CONTROLLER) ps

###############################################################################
# • data_simulator (local) ##
###############################################################################
simulator-up: ## start data_simulator stack
	$(COMPOSE_SIMULATOR) up -d

simulator-down: ## stop data_simulator stack
	$(COMPOSE_SIMULATOR) down --remove-orphans

simulator-restart: ## restart data_simulator stack
	$(COMPOSE_SIMULATOR) restart

simulator-build: ## build data_simulator image
	$(COMPOSE_SIMULATOR) build data_simulator

simulator-rebuild: ## rebuild and restart data_simulator
	$(COMPOSE_SIMULATOR) build data_simulator
	$(COMPOSE_SIMULATOR) up -d

simulator-logs: ## tail data_simulator logs
	$(COMPOSE_SIMULATOR) logs -f

simulator-ps: ## show data_simulator status
	$(COMPOSE_SIMULATOR) ps

###############################################################################
# • gitops (hub deploy) ##
###############################################################################
deploy-rule-up: ## start rule_engine (gitops/hub)
	docker compose -f gitops/rule_engine/compose.yml --project-directory gitops/rule_engine up -d

deploy-rule-down: ## stop rule_engine (gitops/hub)
	docker compose -f gitops/rule_engine/compose.yml --project-directory gitops/rule_engine down --remove-orphans

deploy-controller-up: ## start iot_controller (gitops/hub)
	docker compose -f gitops/iot_controller/compose.yml --project-directory gitops/iot_controller up -d

deploy-controller-down: ## stop iot_controller (gitops/hub)
	docker compose -f gitops/iot_controller/compose.yml --project-directory gitops/iot_controller down --remove-orphans

deploy-simulator-up: ## start data_simulator (gitops/hub)
	docker compose -f gitops/data_simulator/compose.yml --project-directory gitops/data_simulator up -d

deploy-simulator-down: ## stop data_simulator (gitops/hub)
	docker compose -f gitops/data_simulator/compose.yml --project-directory gitops/data_simulator down --remove-orphans

###############################################################################
# • dev utils ##
###############################################################################
migrate: ## run Mongo migrations (rule_engine)
	cd $(ROOT) && MONGO_URI=$(MONGO_URI) MONGO_DB=$(MONGO_DB) MONGO_CONNECT_TIMEOUT=$(MONGO_CONNECT_TIMEOUT) go run ./rule_engine/cmd/migrate

test: ## run rule_engine unit tests
	cd $(ROOT) && go test ./rule_engine/internal/service

smoke-rule: ## run rule_engine end-to-end smoke test
	./scripts/smoke_rule_engine.sh
