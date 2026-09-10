BUILD_DIR = bin
SERVER_APP = aplicacao/servidor/main.go
DRIVER_APP = aplicacao/motorista/main.go
PASSENGER_APP = aplicacao/passageiro/main.go

.PHONY: all help build build-servidor build-motorista build-passageiro run-servidor run-motorista run-passageiro test clean docker-up docker-down

all: help

help:
	@echo "                MAKEFILE - VAIJUNTO CARONAS                     "
	@echo
	@echo "Comandos disponíveis:"
	@echo "  make run-servidor      - Inicia o servidor"
	@echo "  make run-motorista     - Inicia o cliente do Motorista"
	@echo "  make run-passageiro    - Inicia o cliente do Passageiro"
	@echo "  make build             - Compila todos os aplicativos "
	@echo "  make test              - Roda todos os testes do projeto"
	@echo "  make clean             - Remove os binários compilados"
	@echo "  make clear-data        - APAGA o banco de dados"
	@echo "  make docker-up         - Sobe o ambiente usando Docker"
	@echo "  make docker-down       - Derruba o ambiente Docker"
	@echo ""


run-servidor:
	@echo "=> Iniciando Servidor VAIJUNTO"
	go run $(SERVER_APP)

run-motorista:
	@echo "=> Iniciando Módulo do Motorista..."
	go run $(DRIVER_APP)

run-passageiro:
	@echo "=> Iniciando Módulo do Passageiro..."
	go run $(PASSENGER_APP)

# Build)
build: build-servidor build-motorista build-passageiro
	@echo "=> Todos os binários foram gerados na pasta $(BUILD_DIR)/"

build-servidor:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/servidor $(SERVER_APP)

build-motorista:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/motorista $(DRIVER_APP)

build-passageiro:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/passageiro $(PASSENGER_APP)

# Testes e Limpeza
test:
	@echo "=> Executando testes..."
	go test -v ./tests/...

clean:
	@echo "=> Removendo binários..."
	rm -rf $(BUILD_DIR)

clear-data:
	@echo "=> Limpando arquivos de dados JSON e Logs..."
	rm -f dados/*.json dados/*.log

# Docker
docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down