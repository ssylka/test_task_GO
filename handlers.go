package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// Server отвечает на HTTP-запросы, внутри лежит работа с базой (Storage)
type Server struct {
	storage *Storage
}

func NewServer(storage *Storage) *Server {
	return &Server{storage: storage}
}

// handleOperation — обработчик POST /api/v1/wallet (пополнение или списание)
func (s *Server) handleOperation(w http.ResponseWriter, r *http.Request) {
	// Читаем JSON из тела запроса
	var req WalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "не смог разобрать JSON")
		return
	}

	// Проверяем, что walletId — это правильный UUID
	if _, err := uuid.Parse(req.WalletID); err != nil {
		writeError(w, http.StatusBadRequest, "walletId должен быть корректным UUID")
		return
	}

	// Сумма должна быть больше нуля
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount должен быть положительным")
		return
	}

	// Выбираем, что делать: пополнить или списать
	var balance int64
	var err error
	if req.OperationType == OperationDeposit {
		balance, err = s.storage.Deposit(r.Context(), req.WalletID, req.Amount)
	} else if req.OperationType == OperationWithdraw {
		balance, err = s.storage.Withdraw(r.Context(), req.WalletID, req.Amount)
	} else {
		writeError(w, http.StatusBadRequest, "operationType должен быть DEPOSIT или WITHDRAW")
		return
	}

	// Если что-то пошло не так — отдаём подходящую ошибку
	if err != nil {
		s.sendError(w, err)
		return
	}

	// Всё хорошо — отдаём новый баланс
	writeJSON(w, http.StatusOK, BalanceResponse{WalletID: req.WalletID, Balance: balance})
}

// handleBalance — обработчик GET /api/v1/wallets/{walletId}
func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	// Достаём walletId из адреса
	walletID := r.PathValue("walletId")

	if _, err := uuid.Parse(walletID); err != nil {
		writeError(w, http.StatusBadRequest, "walletId должен быть корректным UUID")
		return
	}

	balance, err := s.storage.GetBalance(r.Context(), walletID)
	if err != nil {
		s.sendError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, BalanceResponse{WalletID: walletID, Balance: balance})
}

// sendError выбирает правильный HTTP-код для ошибки
// Нет кошелька или не хватает денег — это не поломка сервера, а обычная ситуация
func (s *Server) sendError(w http.ResponseWriter, err error) {
	if err.Error() == walletNotFoundMessage {
		writeError(w, http.StatusNotFound, err.Error()) // 404
	} else if err.Error() == notEnoughMoneyMessage {
		writeError(w, http.StatusUnprocessableEntity, err.Error()) // 422
	} else {
		// Сюда попадаем только при настоящей проблеме с базой
		log.Printf("ошибка: %v", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
	}
}

// writeJSON отправляет ответ в виде JSON
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError отправляет ответ вида {"error": "текст"}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
