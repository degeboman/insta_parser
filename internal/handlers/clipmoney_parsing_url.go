package handlers

import (
	"encoding/json"
	"inst_parser/internal/usecase/parsing_urls"
	"log/slog"
	"net/http"

	"inst_parser/internal/models"
)

type (
	ClipMoneyParsingUrlRequest struct {
		Url string `json:"url"`
	}
	ClipMoneyParsingUrlResponse struct {
		Success bool              `json:"success"`
		Message string            `json:"message"`
		Data    *models.ResultRow `json:"data"`
	}
)

type ClipMoneyParsingUrl struct {
	logger  *slog.Logger
	usecase *parsing_urls.Usecase
}

func NewClipMoneyParsingUrl(logger *slog.Logger, usecase *parsing_urls.Usecase) *ClipMoneyParsingUrl {
	return &ClipMoneyParsingUrl{logger: logger, usecase: usecase}
}

func (h *ClipMoneyParsingUrl) ClipMoneyParsingUrl(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON из тела запроса
	var req ClipMoneyParsingUrlRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp := ClipMoneyParsingUrlResponse{
			Success: false,
			Message: "Invalid JSON format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Проверяем, что tablename передан
	if req.Url == "" {
		resp := ClipMoneyParsingUrlResponse{
			Success: false,
			Message: "url is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	data, err := h.usecase.ClipMoneyParseUrl(
		req.Url,
	)
	if err != nil {
		h.logger.Error("Failed to parse account data",
			slog.String("url", req.Url),
			slog.String("err", err.Error()),
		)

		resp := ClipMoneyParsingUrlResponse{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Возвращаем успешный ответ
	resp := ClipMoneyParsingUrlResponse{
		Success: true,
		Message: "",
		Data:    data,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
