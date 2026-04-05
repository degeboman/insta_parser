package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inst_parser/internal/models"
	"inst_parser/internal/usecase/parsing_urls"
)

type (
	// ClipMoneyParsingUrlRequest represents the request body for parsing clip, video or reel
	ClipMoneyParsingUrlRequest struct {
		Url string `json:"url" example:"https://www.youtube.com/shorts/2EMmfcZ_UuY"` // URL to parse
	}

	// ClipMoneyParsingUrlResponse represents the response structure for URL parsing
	ClipMoneyParsingUrlResponse struct {
		Success bool                 `json:"success" example:"true"`                    // Operation success status
		Message string               `json:"message" example:"URL parsed successfully"` // Response message
		Data    *models.ResultRowUrl `json:"data"`                                      // Parsed video data
	}
)

type ClipMoneyParsingUrl struct {
	logger  *slog.Logger
	usecase *parsing_urls.Usecase
}

func NewClipMoneyParsingUrl(logger *slog.Logger, usecase *parsing_urls.Usecase) *ClipMoneyParsingUrl {
	return &ClipMoneyParsingUrl{logger: logger, usecase: usecase}
}

// ClipMoneyParsingUrl godoc
// @Summary      Parse video by URL
// @Description  Parse video from tiktok, clip from youtube,vk or reel from instagram
// @Tags         ClipMoney
// @Accept       json
// @Produce      json
// @Param        request body ClipMoneyParsingUrlRequest true "URL to parse"
// @Success      200  {object}  ClipMoneyParsingUrlResponse  "Successfully parsed URL"
// @Failure      400  {object}  ClipMoneyParsingUrlResponse  "Invalid request format or missing URL"
// @Failure      405  {object}  ClipMoneyParsingUrlResponse  "Method not allowed"
// @Failure      500  {object}  ClipMoneyParsingUrlResponse  "Internal server error"
// @Router       /clip_money/parsing_url [post]
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
