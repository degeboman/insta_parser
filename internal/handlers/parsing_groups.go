package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inst_parser/internal/models"
)

type (
	ParsingAccountRequest struct {
		SpreadsheetID string `json:"spreadsheet_id"`
		SheetName     string `json:"sheet_name"`
		IsSelected    bool   `json:"is_selected"`
	}
	ParsingAccountResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
)

type ParsingAccount struct {
	logger        *slog.Logger
	queueProvider QueueProvider
}

func NewParsingAccountsHandler(
	log *slog.Logger,
	queueProvider QueueProvider,
) *ParsingAccount {
	return &ParsingAccount{
		logger:        log,
		queueProvider: queueProvider,
	}
}

func (h *ParsingAccount) ParsingAccount(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON из тела запроса
	var req ParsingAccountRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp := ParsingAccountResponse{
			Success: false,
			Message: "Invalid JSON format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Проверяем, что tablename передан
	if req.SpreadsheetID == "" {
		resp := ParsingAccountResponse{
			Success: false,
			Message: "spreadsheet_id is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if req.SheetName == "" {
		resp := ParsingAccountResponse{
			Success: false,
			Message: "sheetname is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	//go h.usecase.ParseAccount(
	//	req.IsSelected,
	//	req.SheetName,
	//	req.SpreadsheetID,
	//)

	if err := h.queueProvider.Enqueue(models.QueueRequest{
		SpreadsheetID: req.SpreadsheetID,
		SheetName:     req.SheetName,
		IsSelected:    req.IsSelected,
		Type:          1,
	}); err != nil {
		h.logger.Error("failed to enqueue spreadsheet item",
			slog.String("spreadsheet_id", req.SpreadsheetID),
			slog.String("err", err.Error()),
		)

		resp := ParsingUrlsResponse{
			Success: false,
			Message: "failed to enqueue spreadsheet item",
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Возвращаем успешный ответ
	resp := ParsingAccountResponse{
		Success: true,
		Message: "ParsingAccountRequest received successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
