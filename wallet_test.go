package main

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// openTestDB подключается к базе для тестов.
// Если база не запущена — тест просто пропускается, а не падает.
func openTestDB(t *testing.T) *Storage {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://wallet:wallet@localhost:5433/wallet?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skip("база недоступна, пропускаем тест")
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skip("база недоступна, пропускаем тест")
	}

	migrate(context.Background(), pool)
	return &Storage{pool: pool}
}

// Тест: пополнили счёт и проверили баланс.
func TestDeposit(t *testing.T) {
	db := openTestDB(t)
	id := uuid.NewString()

	balance, err := db.Deposit(context.Background(), id, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 1000 {
		t.Fatalf("ожидали 1000, получили %d", balance)
	}
}

// Тест: пополнили, потом списали.
func TestWithdraw(t *testing.T) {
	db := openTestDB(t)
	id := uuid.NewString()

	db.Deposit(context.Background(), id, 1000)

	balance, err := db.Withdraw(context.Background(), id, 400)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 600 {
		t.Fatalf("ожидали 600, получили %d", balance)
	}
}

// Тест: нельзя списать больше, чем есть на счёте.
func TestWithdrawNotEnough(t *testing.T) {
	db := openTestDB(t)
	id := uuid.NewString()

	db.Deposit(context.Background(), id, 100)

	_, err := db.Withdraw(context.Background(), id, 1000)
	if err == nil || err.Error() != notEnoughMoneyMessage {
		t.Fatalf("ожидали ошибку про нехватку денег, получили %v", err)
	}
}

// Главный тест: 1000 пополнений ОДНОВРЕМЕННО по одному кошельку.
// В конце баланс должен быть ровно 1000 — значит, ничего не потерялось.
func TestConcurrentDeposits(t *testing.T) {
	db := openTestDB(t)
	id := uuid.NewString()

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Deposit(context.Background(), id, 1)
		}()
	}
	wg.Wait()

	balance, _ := db.GetBalance(context.Background(), id)
	if balance != 1000 {
		t.Fatalf("ожидали 1000, получили %d", balance)
	}
}
