package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inst_parser/internal/constants"
	"inst_parser/internal/models"
)

type (
	ParsingVkGroupsRequest struct {
		SpreadsheetID string `json:"spreadsheet_id"`
		SheetName     string `json:"sheet_name"`
		IsSelected    bool   `json:"is_selected"`
	}
	ParsingVkGroupsResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
)

type (
	GroupsUrlsProvider interface {
		GroupsUrls(
			isSelected bool,
			sheetName, spreadsheetID string,
		) ([]*models.UrlInfo, error)
	}

	OwnerIDsProvider interface {
		OwnerIDsByGroupsUrls(groupsUrls []*models.UrlInfo) ([]*models.GroupInfoPair, error)
	}

	ClipsInfoProvider interface {
		GetClipsInfoByOwnerID(groupInfo *models.GroupInfoPair) ([]*models.ClipInfo, error)
	}

	GroupsDataInserter interface {
		InsertGroupsData(spreadsheetID, sheetName string, data []*models.ClipInfo) error
	}

	TrackerService interface {
		EnsureProgressSheet(spreadsheetID string) error
		StartParsing(spreadsheetID string, totalURLs int) (int, error)
		UpdateProgress(spreadsheetID string, row, progress int) error
		FinishParsing(spreadsheetID string, row int) error
	}
)

type ParsingVkGroupsHandler struct {
	logger             *slog.Logger
	parser             Parser
	dataInserter       GroupsDataInserter
	groupsUrlsProvider GroupsUrlsProvider
	ownersIDsProvider  OwnerIDsProvider
	clipsInfoProvider  ClipsInfoProvider
	trackerService     TrackerService
}

func NewParsingVkGroupsHandler(
	log *slog.Logger,
	dataInserter GroupsDataInserter,
	groupsUrlsProvider GroupsUrlsProvider,
	ownerIDsProvider OwnerIDsProvider,
	clipsInfoProvider ClipsInfoProvider,
	trackerService TrackerService,
) *ParsingVkGroupsHandler {
	return &ParsingVkGroupsHandler{
		logger:             log,
		dataInserter:       dataInserter,
		groupsUrlsProvider: groupsUrlsProvider,
		ownersIDsProvider:  ownerIDsProvider,
		clipsInfoProvider:  clipsInfoProvider,
		trackerService:     trackerService,
	}
}

const (
	maxCount     = 10000
	defaultCount = 10
)

func (h *ParsingVkGroupsHandler) ParsingVkGroups(w http.ResponseWriter, r *http.Request) {
	// Разрешаем только POST метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON из тела запроса
	var req ParsingVkGroupsRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp := ParsingVkGroupsResponse{
			Success: false,
			Message: "Invalid JSON format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Проверяем, что tablename передан
	if req.SpreadsheetID == "" {
		resp := ParsingVkGroupsResponse{
			Success: false,
			Message: "spreadsheet_id is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if req.SheetName == "" {
		resp := ParsingVkGroupsResponse{
			Success: false,
			Message: "sheetname is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	go h.parsingVkGroups(
		req.IsSelected,
		req.SheetName,
		req.SpreadsheetID,
	)

	// Возвращаем успешный ответ
	resp := ParsingVkGroupsResponse{
		Success: true,
		Message: "ParsingVkGroupsRequest received successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *ParsingVkGroupsHandler) parsingVkGroups(
	isSelected bool,
	sheetName, spreadsheetID string,
) {
	h.logger.Info("ParsingVkGroups request started",
		slog.String("spreadsheet_id", spreadsheetID),
		slog.String("sheet_name", sheetName),
	)
	defer h.logger.Info("ParsingVkGroups request finished",
		slog.String("spreadsheet_id", spreadsheetID),
		slog.String("sheet_name", sheetName),
	)

	groupsUrls, err := h.groupsUrlsProvider.GroupsUrls(isSelected, sheetName, spreadsheetID)
	if err != nil {
		h.logger.Error("Failed to find groups urls",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", sheetName),
			slog.String("err", err.Error()),
		)
		return
	}

	if len(groupsUrls) == 0 {
		h.logger.Warn("Not found urls for parsing",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", sheetName),
		)
		return
	}

	h.logger.Info("Find groups urls successfully",
		slog.Int("count", len(groupsUrls)),
		slog.String("spreadsheet_id", spreadsheetID),
		slog.String("sheet_name", sheetName),
	)

	groupsInfo, err := h.ownersIDsProvider.OwnerIDsByGroupsUrls(groupsUrls)
	if err != nil {
		h.logger.Error("Failed to get owner ids for groups",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", sheetName),
			slog.String("err", err.Error()),
		)
		return
	}

	if len(groupsInfo) == 0 {
		h.logger.Warn("Not found groups info for parsing",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", sheetName),
		)
	}

	if len(groupsInfo) != len(groupsUrls) {
		h.logger.Warn("Count groups info does not match groups urls",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("spreadsheet_id", spreadsheetID),
			slog.Int("count_groups_info", len(groupsInfo)),
			slog.Int("count_group_ids", len(groupsUrls)),
		)
	}

	if err := h.trackerService.EnsureProgressSheet(spreadsheetID); err != nil {
		h.logger.Error("failed to ensure progress sheet",
			slog.String("spreadsheet_id", spreadsheetID),
		)
	}

	progressRow, errStartParsing := h.trackerService.StartParsing(spreadsheetID, len(groupsInfo))
	if errStartParsing != nil {
		h.logger.Error("Error starting progress tracking", errStartParsing)
	}

	defer func() {
		if err := h.trackerService.FinishParsing(spreadsheetID, progressRow); err != nil {
			h.logger.Error("Error finishing progress tracking", err)
		}
	}()

	var processedCount int
	for _, info := range groupsInfo {
		if info.Count <= 0 {
			info.Count = defaultCount
		}

		if info.Count > maxCount {
			info.Count = maxCount
		}

		result, getClipsInfoErr := h.clipsInfoProvider.GetClipsInfoByOwnerID(info)
		if getClipsInfoErr != nil {
			h.logger.Error("Failed to get clips info",
				slog.String("owner_id", info.OwnerID),
				slog.String("spreadsheet_id", spreadsheetID),
				slog.String("sheet_name", sheetName),
				slog.String("err", getClipsInfoErr.Error()),
			)

			continue
		}

		if insertErr := h.dataInserter.InsertGroupsData(spreadsheetID, constants.VkGroupsDataTable, result); insertErr != nil {
			h.logger.Error("Failed to insert data",
				slog.String("owner_id", info.OwnerID),
				slog.String("spreadsheet_id", spreadsheetID),
				slog.String("sheet_name", sheetName),
				slog.String("err", insertErr.Error()),
			)

			continue
		}

		processedCount++
		if err := h.trackerService.UpdateProgress(spreadsheetID, progressRow, processedCount); err != nil {
			h.logger.Error("Error updating progress", err)
		}
	}
}
