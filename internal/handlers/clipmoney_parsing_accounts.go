package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inst_parser/internal/models"
	"inst_parser/internal/usecase/parsing_account"
)

type (
	// ClipMoneyParsingAccountRequest represents the request body for parsing account
	ClipMoneyParsingAccountRequest struct {
		AccountUrl string `json:"account_url" example:"https://vk.ru/id41699827"` // Account URL
	}

	// ClipMoneyParsingAccountResponse represents the response structure
	ClipMoneyParsingAccountResponse struct {
		Success bool                         `json:"success" example:"true"`    // Response success status
		Message string                       `json:"message" example:"success"` // Response message
		Data    []*models.ClipMoneyResultRow `json:"data"`                      // Parsed account data
	}
)

type ClipMoneyParsingAccount struct {
	logger  *slog.Logger
	usecase *parsing_account.Usecase
}

func NewClipMoneyParsingAccount(logger *slog.Logger, usecase *parsing_account.Usecase) *ClipMoneyParsingAccount {
	return &ClipMoneyParsingAccount{logger: logger, usecase: usecase}
}

// ClipMoneyParsingAccount godoc
// @Summary      Parses clips, reels, videos for an account
// @Description  Parses clips for youtube, vk account, videos for tiktok and reels for instagram
// @Tags         ClipMoney
// @Accept       json
// @Produce      json
// @Param        request body ClipMoneyParsingAccountRequest true "Account URL to parse"
// @Success      200  {object}  ClipMoneyParsingAccountResponse  "Successfully parsed account"
// @Failure      400  {object}  ClipMoneyParsingAccountResponse  "Invalid request format or missing account_url"
// @Failure      405  {object}  ClipMoneyParsingAccountResponse  "Method not allowed"
// @Failure      500  {object}  ClipMoneyParsingAccountResponse  "Internal server error"
// @Router       /clip_money/parsing_account [post]
func (h *ClipMoneyParsingAccount) ClipMoneyParsingAccount(w http.ResponseWriter, r *http.Request) {
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
		resp := ClipMoneyParsingAccountResponse{
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

		resp := ClipMoneyParsingAccountResponse{
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
