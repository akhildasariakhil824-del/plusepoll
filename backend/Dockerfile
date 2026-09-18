# Build Stage
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server .

# Production Runtime Stage
FROM alpine:3.20
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/server /app/server

EXPOSE 8080
ENV GIN_MODE=release
ENTRYPOINT ["/app/server"]
