# Wallet Service

Небольшой HTTP-сервис на Go для пополнения и списания средств с кошелька.
Главная задача — корректно работать под высокой параллельной нагрузкой
(1000 запросов в секунду по одному кошельку) и не отдавать ошибки 5xx.

Стек: **Go 1.22**, **PostgreSQL 16**, **Docker / docker-compose**.

## Запуск

Вся система (приложение + база) поднимается одной командой:

```bash
docker compose up --build
```

Сервис будет доступен на `http://localhost:8080`.

Остановить и удалить данные:

```bash
docker compose down -v
```

## API

### Пополнение / списание

```
POST /api/v1/wallet
Content-Type: application/json

{
  "walletId": "8f1d2c3e-...-uuid",
  "operationType": "DEPOSIT",   // или "WITHDRAW"
  "amount": 1000
}
```

Ответ `200 OK`:

```json
{ "walletId": "8f1d2c3e-...-uuid", "balance": 1000 }
```

`amount` — целое положительное число (в минимальных единицах, например в копейках).
Пополнение (`DEPOSIT`) само создаёт кошелёк при первом обращении.
Списание (`WITHDRAW`) требует существующий кошелёк с достаточным балансом.

### Получение баланса

```
GET /api/v1/wallets/{walletId}
```

Ответ `200 OK`:

```json
{ "walletId": "8f1d2c3e-...-uuid", "balance": 600 }
```

### Коды ответов

| Ситуация | Код |
|---|---|
| Успех | `200` |
| Неверный JSON / UUID / operationType / неположительная сумма | `400` |
| Кошелёк не найден | `404` |
| Недостаточно средств | `422` |
| Реальная ошибка базы | `500` |

Ситуации, связанные с конкурентной нагрузкой, никогда не приводят к 5xx.

## Пример запросов (curl)

```bash
WID=$(uuidgen)

# Пополнить на 1000
curl -s -X POST localhost:8080/api/v1/wallet \
  -H 'Content-Type: application/json' \
  -d "{\"walletId\":\"$WID\",\"operationType\":\"DEPOSIT\",\"amount\":1000}"

# Списать 250
curl -s -X POST localhost:8080/api/v1/wallet \
  -H 'Content-Type: application/json' \
  -d "{\"walletId\":\"$WID\",\"operationType\":\"WITHDRAW\",\"amount\":250}"

# Узнать баланс
curl -s localhost:8080/api/v1/wallets/$WID
```

## Как решена проблема конкурентности

Это главный пункт задачи. Самая частая ошибка — «потерянное обновление»:
если прочитать баланс в коде, прибавить в памяти и записать обратно, то два
параллельных запроса затрут изменения друг друга.

Чтобы этого избежать, всё изменение баланса делается **одним SQL-запросом**:

Пополнение — создать-или-прибавить за одно атомарное действие:

```sql
INSERT INTO wallets (id, balance) VALUES ($1, $2)
ON CONFLICT (id) DO UPDATE
    SET balance = wallets.balance + EXCLUDED.balance
RETURNING balance;
```

Списание — проверка «хватает ли денег» и само списание неделимы благодаря
условию `balance >= $1`:

```sql
UPDATE wallets SET balance = balance - $1
WHERE id = $2 AND balance >= $1
RETURNING balance;
```

PostgreSQL на время `UPDATE` блокирует строку, поэтому параллельные запросы по
одному кошельку встают в очередь и выполняются по очереди — без потерянных
обновлений и без ошибок. Дополнительно ограничение `CHECK (balance >= 0)` не даёт
балансу уйти в минус. Пул соединений задаёт размер очереди под нагрузкой.

## Тесты

Юнит-тесты используют заглушку хранилища в памяти, поэтому база не нужна:

```bash
go mod tidy   # один раз, чтобы скачать зависимости
go test       # запустит все тесты
```

Среди них `TestConcurrentDeposits` — 1000 одновременных пополнений по одному
кошельку через весь HTTP-слой: проверяет, что все запросы вернули 200 и баланс
получился ровно 1000.

Есть и тест с настоящей базой `TestConcurrentDepositsRealDB` — он проверяет
реальную защиту от гонок в PostgreSQL и запускается только если задать адрес базы:

```bash
# база поднята через docker compose up db
TEST_DATABASE_URL='postgres://wallet:wallet@localhost:5432/wallet?sslmode=disable' go test -run RealDB
```

## Файлы проекта

```
main.go            точка входа: конфиг, подключение к БД, маршруты, запуск сервера
config.go          чтение настроек из config.env
models.go          типы запроса/ответа и ошибки
storage.go         работа с PostgreSQL (атомарные запросы — защита от гонок)
handlers.go        HTTP-обработчики
wallet_test.go     тесты, включая проверку конкурентности
schema.sql         схема таблицы (применяется при старте)
config.env         переменные окружения
Dockerfile         сборка образа приложения
docker-compose.yml поднимает приложение и базу вместе
```
