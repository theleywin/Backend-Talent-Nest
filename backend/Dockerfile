# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Instalar dependencias necesarias para compilar con CGO en Alpine
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build con CGO habilitado para SQLite (arquitectura nativa)
RUN CGO_ENABLED=1 go build -o talentnest main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /root/

# Create directory for database inside container
RUN mkdir -p /root/data

# Copy the built binary from builder
COPY --from=builder /app/talentnest .

# Database will be stored in container (ephemeral)
ENV DB_PATH=/root/data/talentnest.db

# Expose the port
EXPOSE 3000

# Run the binary
CMD ["./talentnest"]