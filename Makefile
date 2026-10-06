.PHONY: dev build test vet fmt tidy web-install web-build swag docker-build docker-up docker-down help

BIN := bin/server
ifeq ($(OS),Windows_NT)
	BIN := bin/server.exe
endif

help: ## Tampilkan daftar target
	@echo "Targets: dev build test vet fmt tidy web-install web-build swag docker-build docker-up docker-down"

dev: ## Jalankan server (auto migrate; seed bila SEED_ON_START=true)
	go run ./cmd/server

build: ## Build binary ke bin/
	go build -o $(BIN) ./cmd/server

test: ## Jalankan seluruh test (butuh MySQL, lihat .env)
	go test ./... -count=1

vet: ## go vet
	go vet ./...

fmt: ## gofmt seluruh source
	gofmt -w .

tidy:
	go mod tidy

web-install: ## Install dependensi SPA admin
	cd web/admin && npm ci

web-build: ## Build SPA admin ke web/dist
	cd web/admin && npm run build

swag: ## Generate dokumentasi Swagger ke docs/
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs --parseInternal

docker-build:
	docker build -t absensi-go .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down
