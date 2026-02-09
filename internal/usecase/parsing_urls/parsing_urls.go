package parsing_urls

import (
	"log/slog"
	"time"

	"inst_parser/internal/constants"
	"inst_parser/internal/models"
)

type Usecase struct {
	logger                    *slog.Logger
	urlsProvider              UrlsProvider
	trackerService            TrackerService
	dataInserter              DataInserter
	instagramReelInfoProvider InstagramReelInfoProvider
	vkClipInfoProvider        VKClipInfoProvider
	youtubeShortInfoProvider  YoutubeShortInfoProvider
}

func NewUsecase(
	logger *slog.Logger,
	urlsProvider UrlsProvider,
	dataInserter DataInserter,
	instagramReelInfoProvider InstagramReelInfoProvider,
	vkClipInfoProvider VKClipInfoProvider,
	trackerService TrackerService,
	youtubeShortInfoProvider YoutubeShortInfoProvider,
) *Usecase {
	return &Usecase{
		logger:                    logger,
		urlsProvider:              urlsProvider,
		dataInserter:              dataInserter,
		instagramReelInfoProvider: instagramReelInfoProvider,
		vkClipInfoProvider:        vkClipInfoProvider,
		trackerService:            trackerService,
		youtubeShortInfoProvider:  youtubeShortInfoProvider,
	}
}

type (
	UrlsProvider interface {
		FindUrls(
			isSelected bool,
			parsingTypes []models.ParsingType,
			sheetName, spreadsheetID string,
		) ([]*models.UrlInfo, error)
	}

	TrackerService interface {
		EnsureProgressSheet(spreadsheetID string) error
		StartParsing(spreadsheetID string, totalURLs int) (int, error)
		UpdateProgress(spreadsheetID string, row, progress int) error
		FinishParsing(spreadsheetID string, row int) error
	}

	VKClipInfoProvider interface {
		ClipInfo(ownerID, clipID int) (*models.VKClipInfo, error)
	}

	InstagramReelInfoProvider interface {
		GetInstagramReelInfo(reelURL string) (*models.InstagramAPIResponse, error)
	}

	YoutubeShortInfoProvider interface {
		YoutubeShortInfo(shortID string) (*models.YoutubeShortInfoApiResponse, error)
	}

	DataInserter interface {
		InsertData(
			spreadsheetID,
			sheetName,
			rangeData string,
			data [][]interface{},
		) error
	}
)

const batchSize = 50

func (u *Usecase) ParseUrls(
	isSelected bool,
	sheetName, spreadsheetID string,
) {
	u.logger.Info("ParseUrls started")
	defer u.logger.Info("ParseUrls finished")

	urls, err := u.urlsProvider.FindUrls(
		isSelected,
		[]models.ParsingType{
			models.InstagramParsingType,
			models.VKParsingType,
			models.YoutubeParsingType,
		},
		sheetName,
		spreadsheetID,
	)
	if err != nil {
		u.logger.Error("Failed to find urls",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", sheetName),
			slog.String("err", err.Error()),
		)
		return
	}

	if len(urls) == 0 {
		u.logger.Warn("Not found urls for parsing",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", sheetName),
		)
		return
	}

	if err = u.trackerService.EnsureProgressSheet(spreadsheetID); err != nil {
		u.logger.Error("Failed to ensure progress sheet",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("err", err.Error()),
		)
	}

	progressRow, errStartParsing := u.trackerService.StartParsing(spreadsheetID, len(urls))
	if errStartParsing != nil {
		u.logger.Error("Error starting progress tracking",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("err", errStartParsing.Error()),
		)
	}
	defer func() {
		if err = u.trackerService.FinishParsing(spreadsheetID, progressRow); err != nil {
			u.logger.Error("Error finishing progress tracking",
				slog.String("err", err.Error()),
			)
		}
	}()

	results := make([]*models.ResultRow, 0, len(urls))

	var processedCount int
	for i := 0; i < len(urls); i += batchSize {
		end := i + batchSize
		if end > len(urls) {
			end = len(urls)
		}

		batch := urls[i:end]
		batchResults := u.processBatchUrl(batch)
		results = append(results, batchResults...)

		processedCount += len(batch)

		if err := u.trackerService.UpdateProgress(spreadsheetID, progressRow, processedCount); err != nil {
			u.logger.Error("Error updating progress",
				slog.String("spreadsheet_id", spreadsheetID),
				slog.String("sheet_name", sheetName),
				slog.String("err", err.Error()),
			)
		}
	}

	if err := u.dataInserter.InsertData(
		spreadsheetID,
		constants.DataTable,
		"A:I",
		models.ResultRowToInterface(results),
	); err != nil {
		u.logger.Error("ParsingUrls URLs returned an error",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", sheetName),
			slog.String("err", err.Error()),
		)
		return
	}
}

func (u *Usecase) processBatchUrl(
	urls []*models.UrlInfo,
) []*models.ResultRow {
	const (
		instaTimeout = 550 * time.Millisecond
		vkTimeout    = 250 * time.Millisecond
	)
	results := make([]*models.ResultRow, len(urls))

	for i, url := range urls {
		var resultRow *models.ResultRow
		switch models.ParsingTypeByUrl(url.URL) {
		case models.InstagramParsingType:
			time.Sleep(instaTimeout)
			resultRow = u.parseInstagram(url.URL)
		case models.VKParsingType:
			time.Sleep(vkTimeout)
			resultRow = u.parseVK(url.URL)
		case models.YoutubeParsingType:
			resultRow = u.parseYoutubeShort(url.URL)
		default:
			u.logger.Warn("Unsupported URL type",
				slog.String("url", url.URL),
			)
			continue
		}

		if resultRow == nil {
			continue
		}
		results[i] = resultRow
	}

	return results
}

func (u *Usecase) parseInstagram(url string) *models.ResultRow {
	data, err := u.instagramReelInfoProvider.GetInstagramReelInfo(url)
	if err != nil {
		u.logger.Warn("Error fetching instagram data",
			slog.String("url", url),
			slog.String("err", err.Error()),
		)

		return models.EmptyResultRow(url)
	}

	resultRow, err := models.ProcessInstagramResponse(data, url)
	if err != nil {
		u.logger.Error("Error processing instagram response",
			slog.String("err", err.Error()),
		)
		return nil
	}

	return resultRow
}

func (u *Usecase) parseVK(url string) *models.ResultRow {
	ownerID, clipID, err := models.ParseVkClipURL(url)
	if err != nil {
		u.logger.Error("Error parsing vk clip url",
			slog.String("url", url),
			slog.String("err", err.Error()),
		)

		return models.EmptyResultRow(url)
	}

	result, err := u.vkClipInfoProvider.ClipInfo(ownerID, clipID)
	if err != nil {
		u.logger.Error("Error getting clip info",
			slog.String("url", url),
			slog.String("err", err.Error()),
		)

		return models.EmptyResultRow(url)
	}

	return models.ProcessVKClipInfoToResultRow(url, result)
}

func (u *Usecase) parseYoutubeShort(url string) *models.ResultRow {
	shortID, ok := models.ExtractYouTubeShortsID(url)
	if !ok {
		u.logger.Error("failed to extract youtube short id from url",
			slog.String("url", url),
		)
		return models.EmptyResultRow(url)
	}

	result, err := u.youtubeShortInfoProvider.YoutubeShortInfo(shortID)
	if err != nil {
		u.logger.Error("Error getting youtube short info",
			slog.String("url", url),
			slog.String("err", err.Error()),
		)

		return models.EmptyResultRow(url)
	}

	return result.ToResultRow()
}
