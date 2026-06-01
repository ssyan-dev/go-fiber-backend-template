FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates textinfo

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api/main.go


FROM alpine:3.19 AS runner

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/api .

COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./api"]