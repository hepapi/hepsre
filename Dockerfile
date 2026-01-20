# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app
RUN apk update --no-cache && apk add gcc alpine-sdk
# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o hep-sre-mini cmd/server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY internal/templates/ ./internal/templates/
COPY --from=builder /app/hep-sre-mini .
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./hep-sre-mini"]
