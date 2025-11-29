# Build stage
FROM golang:latest AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build for ARM64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o talentnest main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs libc6-compat
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