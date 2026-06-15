package main

import (
	"context"
	"log"
	"net/http"
)

// main — с этого начинается программа
func main() {
	// Читаем настройки из config.env
	cfg := loadConfig("config.env")
	ctx := context.Background()
	// Подключаемся к базе
	pool, err := newPool(ctx, cfg)
	if err != nil {
		log.Fatalf("не смог подключиться к базе: %v", err)
	}
	defer pool.Close()

	// Создаём таблицу, если её ещё нет
	if err := migrate(ctx, pool); err != nil {
		log.Fatalf("ошибка миграции: %v", err)
	}

	// Создаём наш сервер
	storage := &Storage{pool: pool}
	server := NewServer(storage)

	// Привязываем адреса к обработчикам
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/wallet", server.handleOperation)
	mux.HandleFunc("GET /api/v1/wallets/{walletId}", server.handleBalance)

	// Запускаем сервер
	log.Printf("сервер запущен на порту %s", cfg.AppPort)
	if err := http.ListenAndServe(":"+cfg.AppPort, mux); err != nil {
		log.Fatalf("сервер остановился: %v", err)
	}
}
