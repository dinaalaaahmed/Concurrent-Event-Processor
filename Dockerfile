# STAGE 1: SQLC Code Generation using Go image
FROM golang:1.25-alpine AS sqlc-gen
WORKDIR /app

# Install sqlc
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

COPY sqlc.yaml .
COPY internal/adapters/postgresql ./internal/adapters/postgresql
# 3. SQLC generates the Go code directly into the /sqlc subfolder
RUN sqlc generate

# STAGE 2: Build the Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Install goose for migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy your source code
COPY . .

# 4. Grab the generated files from Stage 1 
# This places db.go, models.go, etc., back into your internal adapters
COPY --from=sqlc-gen /app/internal/adapters/postgresql/sqlc ./internal/adapters/postgresql/sqlc

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/

# STAGE 3: Final Production Image
FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /go/bin/goose /usr/local/bin/goose

# 5. Copy the migrations so Goose can find them
COPY --from=builder /app/internal/adapters/postgresql/migrations ./migrations

EXPOSE 3000

# Run migrations then start the app
# Note: The path here points to the root-level ./migrations we just copied
ENTRYPOINT ["sh", "-c", "goose -dir ./migrations postgres \"$DB_DSN\" up && ./main"]