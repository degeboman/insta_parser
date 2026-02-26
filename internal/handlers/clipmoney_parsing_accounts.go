package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inst_parser/internal/models"
	"inst_parser/internal/usecase/parsing_account"
)

type (
	ClipMoneyParsingAccountRequest struct {
		AccountUrl string `json:"account_url"`
	}
	ClipMoneyParsingAccountResponse struct {
		Success bool                         `json:"success"`
		Message string                       `json:"message"`
		Data    []*models.ClipMoneyResultRow `json:"data"`
	}
)

type ClipMoneyParsingAccount struct {
	logger  *slog.Logger
	usecase *parsing_account.Usecase
}

func (h *ParsingAccount) ClipMoneyParsingAccount(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON из тела запроса
	var req ClipMoneyParsingAccountRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp := ClipMoneyParsingAccountResponse{
			Success: false,
			Message: "Invalid JSON format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Проверяем, что tablename передан
	if req.AccountUrl == "" {
		resp := ParsingAccountResponse{
			Success: false,
			Message: "spreadsheet_id is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	data, err := h.usecase.ClipMoneyParseAccount(
		req.AccountUrl,
	)
	if err != nil {
		h.logger.Error("Failed to parse account data",
			slog.String("account_url", req.AccountUrl),
			slog.String("err", err.Error()),
		)

		resp := ParsingAccountResponse{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Возвращаем успешный ответ
	resp := ClipMoneyParsingAccountResponse{
		Success: true,
		Message: "",
		Data:    data,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
