.PHONY: dev build build-frontend clean

# Run in dev mode — serves frontend/dist from disk (build-frontend first)
dev:
	go run . --port 7331

# Build frontend assets into frontend/dist
build-frontend:
	cd frontend && npm install && npm run build

# Build a single self-contained binary with embedded frontend
build: build-frontend
	go build -tags embed -o tracer .

clean:
	rm -rf frontend/dist frontend/node_modules tracer
