BUILD_DIR = bin
SERVER_APP = aplicacao/servidor/main.go
DRIVER_APP = aplicacao/motorista/main.go
PASSENGER_APP = aplicacao/passageiro/main.go

GO = go

.PHONY: all help build build-servidor build-motorista build-passageiro run-servidor run-motorista run-passageiro run-testes test clean clean-data up down

all: help

help:
	@echo "                MAKEFILE - VAIJUNTO CARONAS                     "
	@echo
	@echo "Comandos disponíveis:"
	@echo "  make run-servidor      - Inicia o servidor"
	@echo "  make run-motorista     - Inicia o cliente do Motorista"
	@echo "  make run-passageiro    - Inicia o cliente do Passageiro"
	@echo "  make run-testes        - Roda todos os testes do projeto"
	@echo "  make build             - Compila todos os aplicativos "
	@echo "  make clean             - Remove os binários compilados"
	@echo "  make clean-data        - APAGA o banco de dados"
	@echo "  make up                - Sobe o ambiente usando Docker"
	@echo "  make down              - Derruba o ambiente Docker"
	@echo

run-servidor:
	@echo "=> Iniciando Servidor VAIJUNTO"
	$(GO) run $(SERVER_APP)

run-motorista:
	@echo "=> Iniciando Módulo do Motorista..."
	$(GO) run $(DRIVER_APP)

run-passageiro:
	@echo "=> Iniciando Módulo do Passageiro..."
	$(GO) run $(PASSENGER_APP)

# Build
build: build-servidor build-motorista build-passageiro
	@echo "=> Todos os binários foram gerados na pasta $(BUILD_DIR)/"

build-servidor:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/servidor $(SERVER_APP)

build-motorista:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/motorista $(DRIVER_APP)

build-passageiro:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/passageiro $(PASSENGER_APP)

# Testes e Limpeza
run-testes:
	@echo "=> Executando testes..."
	$(GO) test -v ./tests/...

test: run-testes

clean:
	@echo "=> Removendo binários..."
	rm -rf $(BUILD_DIR)

clean-data:
	@echo "=> Limpando arquivos de dados JSON e Logs..."
	rm -f dados/*.json dados/*.log

# Docker
up:
	docker compose up --build -d

down:
	docker compose down