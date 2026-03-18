.PHONY: build dev frontend backend clean

# Build everything
build: frontend backend

# Build frontend
frontend:
	cd web && npm install && npm run build
	rm -rf cmd/server/dist
	cp -r web/dist cmd/server/dist

# Build backend (requires frontend to be built first)
backend:
	CGO_ENABLED=1 go build -o claude-code-proxy ./cmd/server

# Development: run frontend dev server
dev-frontend:
	cd web && npm run dev

# Development: run backend
dev-backend:
	go run ./cmd/server

# Clean build artifacts
clean:
	rm -f claude-code-proxy
	rm -rf cmd/server/dist
	rm -rf web/dist

# Docker build
docker:
	docker-compose build

# Docker run
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
