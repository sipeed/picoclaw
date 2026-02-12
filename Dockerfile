FROM golang:1.25.7

WORKDIR /app

# Copy go.mod and go.sum first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the app (assuming main.go is in the root)
RUN go build -tags netgo -ldflags "-s -w" -o app ./...

CMD ["./app"]
