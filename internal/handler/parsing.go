package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inst_parser/internal/rapid"
)

type Request struct {
	SpreadsheetID string   `json:"spreadsheet_id"`
	URLs          []string `json:"urls"`
}

type Response struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Data    []string `json:"data"`
}

type Parser interface {
	ParseUrl(reelUrl []string) []*rapid.ResultRow
}

type DataInserter interface {
	InsertData(spreadsheetID, sheetName string, data []*rapid.ResultRow) error
}

type ParsingHandler struct {
	logger       *slog.Logger
	parser       Parser
	dataInserter DataInserter
}

func NewParsingHandler(log *slog.Logger, parser Parser, inserter DataInserter) *ParsingHandler {
	return &ParsingHandler{
		logger:       log,
		parser:       parser,
		dataInserter: inserter,
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
		result := h.parser.ParseUrl(req.URLs)
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
