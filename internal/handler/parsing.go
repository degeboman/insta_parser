package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inst_parser/internal/models"
)

type Request struct {
	SpreadsheetID string   `json:"spreadsheet_id"`
	URLs          []string `json:"urls"`
}

type RequestV2 struct {
	SpreadsheetID string `json:"spreadsheet_id"`
	SheetName     string `json:"sheet_name"`
	IsSelected    bool   `json:"is_selected"`
}

type Response struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Data    []string `json:"data"`
}

type Parser interface {
	ParseUrl(spreadsheetID string, reelUrl []string) []*models.ResultRow
}

type DataInserter interface {
	InsertData(spreadsheetID, sheetName string, data []*models.ResultRow) error
}

type UrlsProvider interface {
	FindUrls(isSelected bool, parsingTypes []models.ParsingType, sheetName, spreadsheetID string) ([]string, error)
}

type ParsingHandler struct {
	logger       *slog.Logger
	parser       Parser
	dataInserter DataInserter
	urlsProvider UrlsProvider
}

func NewParsingHandler(
	log *slog.Logger,
	parser Parser,
	inserter DataInserter,
	urlsProvider UrlsProvider,
) *ParsingHandler {
	return &ParsingHandler{
		logger:       log,
		parser:       parser,
		dataInserter: inserter,
		urlsProvider: urlsProvider,
	}
}

func (h *ParsingHandler) Parsing(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON из тела запроса
	var req Request
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp := Response{
			Success: false,
			Message: "Invalid JSON format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Проверяем, что tablename передан
	if req.SpreadsheetID == "" {
		resp := Response{
			Success: false,
			Message: "spreadsheet_id is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(req.URLs) == 0 {
		resp := Response{
			Success: false,
			Message: "Empty URLs",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	go func() {
		result := h.parser.ParseUrl(req.SpreadsheetID, req.URLs)
		if result == nil || len(result) == 0 {
			h.logger.Warn("Parsing URLs returned an empty result")
			return
		}

		if err := h.dataInserter.InsertData(req.SpreadsheetID, "Сырые данные", result); err != nil {
			h.logger.Error("Parsing URLs returned an error",
				slog.String("spreadsheet_id", req.SpreadsheetID),
				slog.String("err", err.Error()),
			)
			return
		}
	}()

	// Возвращаем успешный ответ
	resp := Response{
		Success: true,
		Message: "Request received successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *ParsingHandler) Parsing2(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON из тела запроса
	var req RequestV2
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp := Response{
			Success: false,
			Message: "Invalid JSON format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Проверяем, что tablename передан
	if req.SpreadsheetID == "" {
		resp := Response{
			Success: false,
			Message: "spreadsheet_id is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if req.SheetName == "" {
		resp := Response{
			Success: false,
			Message: "sheetname is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	go func() {
		h.logger.Info("Parsing2 request started")
		defer h.logger.Info("Parsing2 request finished")

		urls, err := h.urlsProvider.FindUrls(req.IsSelected, []models.ParsingType{models.Instagram}, req.SheetName, req.SpreadsheetID)
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
			h.logger.Warn("Parsing URLs returned an empty result")
			return
		}
		h.logger.Info("ParseUrl finished")

		h.logger.Info("InsertData started")
		if err := h.dataInserter.InsertData(req.SpreadsheetID, "Сырые данные", result); err != nil {
			h.logger.Error("Parsing URLs returned an error",
				slog.String("spreadsheet_id", req.SpreadsheetID),
				slog.String("err", err.Error()),
			)
			return
		}
		h.logger.Info("InsertData finished")
	}()

	// Возвращаем успешный ответ
	resp := Response{
		Success: true,
		Message: "Request received successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
