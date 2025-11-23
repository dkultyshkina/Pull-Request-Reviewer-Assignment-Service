FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o ./bin/reviewer-service \
    ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata && update-ca-certificates
WORKDIR /app

COPY --from=builder /app/bin/reviewer-service .

RUN adduser -D -s /bin/sh reviewer
RUN chown -R reviewer:reviewer /app
USER reviewer

EXPOSE 8080

CMD ["./reviewer-service"]