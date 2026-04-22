# DEPENDENCIES
FROM golang:1.26.2-alpine3.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# SOURCE CODE
COPY . ./
RUN go build -o bin/app cmd/main.go

# FINAL STAGE
FROM alpine AS final
COPY --from=builder /app/migrations /migrations
COPY --from=builder /app/bin/app /app
COPY --from=builder /app/.env /.env

EXPOSE 8080
CMD ["/app"]