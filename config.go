package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config — все настройки приложения
type Config struct {
	AppPort    string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBMaxConns int32
}

// loadConfig читает config.env и заполняет настройки
func loadConfig(path string) *Config {
	// Загружаем строки из файла в переменные окружения
	loadEnvFile(path)

	// DB_MAX_CONNS — это число, переводим строку в int
	maxConns, _ := strconv.Atoi(os.Getenv("DB_MAX_CONNS"))

	return &Config{
		AppPort:    os.Getenv("APP_PORT"),
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBMaxConns: int32(maxConns),
	}
}

// DSN собирает строку подключения к PostgreSQL
func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

// loadEnvFile читает файл config.env и кладёт строки KEY=VALUE в окружение
func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Делим строку на ключ и значение по знаку "="
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		os.Setenv(strings.TrimSpace(key), strings.TrimSpace(value))
	}
}
