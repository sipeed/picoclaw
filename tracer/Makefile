.PHONY: dev build build-frontend clean

# Build frontend assets into frontend/dist (required before running)
build-frontend:
	cd frontend && npm install && npm run build

# Run in dev mode (from repo root, after build-frontend)
dev: build-frontend
	go run ./cmd/picoclaw tracer

# Build a self-contained binary with embedded frontend
build: build-frontend
	go build -tags embed -o picoclaw ./cmd/picoclaw

clean:
	rm -rf frontend/dist frontend/node_modules picoclaw
