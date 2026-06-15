package main

// Тип операции: пополнение или списание
type OperationType string

const (
	OperationDeposit  OperationType = "DEPOSIT"
	OperationWithdraw OperationType = "WITHDRAW"
)

const (
	walletNotFoundMessage = "wallet not found"
	notEnoughMoneyMessage = "not enough money"
)

// WalletRequest — тело входящего POST-запроса
// Теги `json:"..."` говорят, как поле называется в JSON
type WalletRequest struct {
	WalletID      string        `json:"walletId"`
	OperationType OperationType `json:"operationType"`
	Amount        int64         `json:"amount"`
}

// BalanceResponse — то, что мы отвечаем клиенту
type BalanceResponse struct {
	WalletID string `json:"walletId"`
	Balance  int64  `json:"balance"`
}
