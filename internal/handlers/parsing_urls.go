package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inst_parser/internal/constants"
	"inst_parser/internal/models"
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

type (
	Parser interface {
		ParseUrl(spreadsheetID string, reelUrl []*models.UrlInfo) []*models.ResultRow
	}
	DataInserter interface {
		InsertData(spreadsheetID, sheetName string, data []*models.ResultRow) error
	}
	UrlsProvider interface {
		FindUrls(isSelected bool, parsingTypes []models.ParsingType, sheetName, spreadsheetID string) ([]*models.UrlInfo, error)
	}
)

type ParsingUrlsHandler struct {
	logger       *slog.Logger
	parser       Parser
	dataInserter DataInserter
	urlsProvider UrlsProvider
}

func NewParsingUrlsHandler(
	log *slog.Logger,
	parser Parser,
	inserter DataInserter,
	urlsProvider UrlsProvider,
) *ParsingUrlsHandler {
	return &ParsingUrlsHandler{
		logger:       log,
		parser:       parser,
		dataInserter: inserter,
		urlsProvider: urlsProvider,
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

	go func() {
		h.logger.Info("ParsingUrls request started")
		defer h.logger.Info("ParsingUrls request finished")

		urls, err := h.urlsProvider.FindUrls(
			req.IsSelected,
			[]models.ParsingType{
				models.Instagram,
				models.VK,
			},
			req.SheetName,
			req.SpreadsheetID,
		)
		if err != nil {
			h.logger.Error("Failed to find urls",
				slog.String("spreadsheet_id", req.SpreadsheetID),
				slog.String("err", err.Error()),
			)
			return
		}

		if len(urls) == 0 {
			h.logger.Warn("Not found urls for parsing")
			return
		}

		h.logger.Info("ParseUrl started")
		result := h.parser.ParseUrl(req.SpreadsheetID, urls)
		if result == nil || len(result) == 0 {
			h.logger.Warn("ParsingUrls URLs returned an empty result")
			return
		}
		h.logger.Info("ParseUrl finished")

		h.logger.Info("InsertData started")
		if err := h.dataInserter.InsertData(req.SpreadsheetID, constants.DataTable, result); err != nil {
			h.logger.Error("ParsingUrls URLs returned an error",
				slog.String("spreadsheet_id", req.SpreadsheetID),
				slog.String("err", err.Error()),
			)
			return
		}
		h.logger.Info("InsertData finished")
	}()

	// Возвращаем успешный ответ
	resp := ParsingUrlsResponse{
		Success: true,
		Message: "ParsingUrlsRequest received successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
