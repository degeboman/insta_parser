package handlers

import (
	"context"
	"encoding/json"
	"inst_parser/internal/models"
	"log/slog"
	"net/http"
)

type ParsingUrlsV2Handler struct {
	logger         *slog.Logger
	queueProvider  QueueProvider
	parserProvider parserProvider
}

func NewParsingUrlsV2Handler(
	logger *slog.Logger,
	queueProvider QueueProvider,
	parserProvider parserProvider,
) *ParsingUrlsV2Handler {
	return &ParsingUrlsV2Handler{
		logger:         logger,
		queueProvider:  queueProvider,
		parserProvider: parserProvider,
	}
}

func (h *ParsingUrlsV2Handler) ParsingUrls(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON из тела запроса
	var req ParsingUrlsRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp := ParsingUrlsResponse{
			Success: false,
			Message: "Invalid JSON format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Проверяем, что tablename передан
	if req.SpreadsheetID == "" {
		resp := ParsingUrlsResponse{
			Success: false,
			Message: "spreadsheet_id is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if req.SheetName == "" {
		resp := ParsingUrlsResponse{
			Success: false,
			Message: "sheetname is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if err := h.queueProvider.Push(
		context.Background(),
		models.QueueRequest{
			SpreadsheetID: req.SpreadsheetID,
			SheetName:     req.SheetName,
			IsSelected:    req.IsSelected,
			Type:          2,
		}); err != nil {
		h.logger.Error("failed to enqueue spreadsheet item",
			slog.String("spreadsheet_id", req.SpreadsheetID),
			slog.String("table", req.SheetName),
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
	resp := ParsingUrlsResponse{
		Success: true,
		Message: "ParsingUrlsRequest received successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
