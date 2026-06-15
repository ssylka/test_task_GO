# ---- Этап 1: сборка ----
# Берём образ с компилятором Go и собираем программу.
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .

# go mod tidy подтянет зависимости и создаст go.sum.
RUN go mod tidy
# Собираем статический бинарник (CGO_ENABLED=0 — без зависимостей от системных библиотек).
RUN CGO_ENABLED=0 go build -o /app/server .

# ---- Этап 2: запуск ----
# Берём маленький образ alpine и кладём в него только готовый бинарник.
FROM alpine:3.20

WORKDIR /app
COPY --from=builder /app/server /app/server
COPY config.env /app/config.env

EXPOSE 8080
CMD ["/app/server"]
