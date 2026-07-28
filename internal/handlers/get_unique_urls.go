package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type GetUniqueUrlsRequest struct {
	SpreadsheetID string `json:"spreadsheet_id"`
}

type GetUniqueUrlsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type UniqueUrlsProvider interface {
	GetUniqueUrls(spreadsheetID string) error
}

type GetUniqueHandler struct {
	logger             *slog.Logger
	uniqueUrlsProvider UniqueUrlsProvider
}

func NewGetUniqueHandler(logger *slog.Logger, uniqueUrlsProvider UniqueUrlsProvider) *GetUniqueHandler {
	return &GetUniqueHandler{logger: logger, uniqueUrlsProvider: uniqueUrlsProvider}
}

func (h *GetUniqueHandler) GetUniqueUrls(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только GET метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON из тела запроса
	var req GetUniqueUrlsRequest
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

	if err := h.uniqueUrlsProvider.GetUniqueUrls(req.SpreadsheetID); err != nil {
		resp := ParsingUrlsResponse{
			Success: false,
			Message: "err with urls",
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

	// Возвращаем успешный ответ
	resp := GetUniqueUrlsResponse{
		Success: true,
		Message: "GetUniqueUrlsResponse received successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
