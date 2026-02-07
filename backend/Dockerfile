# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Ensure dependencies are tidy
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o ppd-api ./cmd/api/main.go

# Run stage
FROM alpine:3.19

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/ppd-api .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./ppd-api"]
