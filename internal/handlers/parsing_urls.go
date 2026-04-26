package handlers

import (
	"encoding/json"
	"inst_parser/internal/models"
	"log/slog"
	"net/http"
)

type ParsingUrlsRequest struct {
	SpreadsheetID string `json:"spreadsheet_id"`
	SheetName     string `json:"sheet_name"`
	IsSelected    bool   `json:"is_selected"`
}

type ParsingUrlsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type QueueProvider interface {
	Enqueue(req models.QueueRequest) error
}

type ParsingUrlsHandler struct {
	logger        *slog.Logger
	queueProvider QueueProvider
}

func NewParsingUrlsHandler(
	logger *slog.Logger,
	queueProvider QueueProvider,
) *ParsingUrlsHandler {
	return &ParsingUrlsHandler{
		logger:        logger,
		queueProvider: queueProvider,
	}
}

func (h *ParsingUrlsHandler) ParsingUrls(w http.ResponseWriter, r *http.Request) {
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

	//go h.usecase.ParseUrls(
	//	req.IsSelected,
	//	req.SheetName,
	//	req.SpreadsheetID,
	//)

	if err := h.queueProvider.Enqueue(models.QueueRequest{
		SpreadsheetID: req.SpreadsheetID,
		SheetName:     req.SheetName,
		IsSelected:    req.IsSelected,
		Type:          0,
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
	resp := ParsingUrlsResponse{
		Success: true,
		Message: "ParsingUrlsRequest received successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
