# Build stage
FROM golang:latest AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o talentnest main.go


# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the built binary from builder
COPY --from=builder /app/talentnest .

# Expose the port (change if your app uses a different port)
EXPOSE 3000

# Set environment variables (override with docker service update or stack deploy)
# ENV PORT=3000
# ENV MONGODB_URI=mongodb://mongo:27017/talentnest
# ENV JWT_SECRET=your_jwt_secret

# Run the binary
CMD ["./talentnest"]
