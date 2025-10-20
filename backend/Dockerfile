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
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the built binary from builder
COPY --from=builder /app/talentnest .

# Expose the port
EXPOSE 3000

# Run the binary
CMD ["./talentnest"]