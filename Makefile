.PHONY: up down logs build test tidy lint fmt curl-create-order

up: ## Start the full local stack (postgres, mongo, localstack, all services, prometheus, grafana)
	docker compose up --build

down: ## Stop and remove the local stack
	docker compose down -v

logs: ## Tail logs from all services
	docker compose logs -f order-service inventory-service notification-service

build: ## Compile all three services
	cd services/order-service && go build ./...
	cd services/inventory-service && go build ./...
	cd services/notification-service && go build ./...

test: ## Run unit tests for all three services
	cd services/order-service && go test ./...
	cd services/inventory-service && go test ./...
	cd services/notification-service && go test ./...

tidy: ## go mod tidy for all three services
	cd services/order-service && go mod tidy
	cd services/inventory-service && go mod tidy
	cd services/notification-service && go mod tidy

lint: ## go vet for all three services (swap in golangci-lint if you install it)
	cd services/order-service && go vet ./...
	cd services/inventory-service && go vet ./...
	cd services/notification-service && go vet ./...

fmt: ## gofmt all three services
	cd services/order-service && gofmt -l -w .
	cd services/inventory-service && gofmt -l -w .
	cd services/notification-service && gofmt -l -w .

curl-create-order: ## Example request against a running stack (make up, in another shell)
	curl -s -X POST http://localhost:8080/orders \
		-H "Content-Type: application/json" \
		-d '{"customer_id":"cust-1","item_sku":"sku-1","quantity":2}' | jq .
