# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git (required for some Go dependencies)
RUN apk add --no-cache git

# Initialize Go module (router is standalone)
COPY main.go .
RUN go mod init github.com/theleywin/Backend-Talent-Nest/router && \
    go mod tidy

# Build the Go router
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o router main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the built binary from builder
COPY --from=builder /app/router .

# Expose router port
EXPOSE 8080

# Environment variables (can be overridden)
ENV SERVICE_NAME=frontend
ENV SERVICE_PORT=5173
ENV HEALTH_PATH=/
ENV ROUTER_PORT=8080

# Run the router
CMD ["./router"]
