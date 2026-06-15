package main

import (
	"context"
	_ "embed"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Сюда попадёт текст из файла schema.sql
//
//go:embed schema.sql
var schemaSQL string

// Storage — это работа с базой данных, внутри лежит подключение к Postgres
type Storage struct {
	pool *pgxpool.Pool
}

// newPool открывает подключение к базе
func newPool(ctx context.Context, cfg *Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = cfg.DBMaxConns

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	// Проверяем, что база отвечает
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}

// migrate создаёт таблицу, если её ещё нет
func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, schemaSQL)
	return err
}

// Deposit — пополнение счёта
// Если кошелька нет — он создаётся
// Всё делается одним SQL-запросом, поэтому при 1000 запросах одновременно
// деньги не теряются: база сама выстраивает запросы в очередь
func (s *Storage) Deposit(ctx context.Context, walletID string, amount int64) (int64, error) {
	query := `
		INSERT INTO wallets (id, balance)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
			SET balance = wallets.balance + $2
		RETURNING balance;`

	var balance int64
	err := s.pool.QueryRow(ctx, query, walletID, amount).Scan(&balance)
	return balance, err
}

// Withdraw — списание со счёта
// Условие "balance >= $2" не даёт балансу уйти в минус
func (s *Storage) Withdraw(ctx context.Context, walletID string, amount int64) (int64, error) {
	query := `
		UPDATE wallets
		SET balance = balance - $2
		WHERE id = $1 AND balance >= $2
		RETURNING balance;`

	var balance int64
	err := s.pool.QueryRow(ctx, query, walletID, amount).Scan(&balance)

	// pgx.ErrNoRows значит, что ничего не обновилось
	// Причины две: кошелька нет ИЛИ денег не хватило, проверяем, что именно
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM wallets WHERE id = $1)`, walletID).Scan(&exists)
		if !exists {
			return 0, errors.New(walletNotFoundMessage)
		}
		return 0, errors.New(notEnoughMoneyMessage)
	}
	return balance, err
}

// GetBalance — узнать баланс кошелька
func (s *Storage) GetBalance(ctx context.Context, walletID string) (int64, error) {
	var balance int64
	err := s.pool.QueryRow(ctx, `SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errors.New(walletNotFoundMessage)
	}
	return balance, err
}
