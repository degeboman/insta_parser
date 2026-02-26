package parsing_account

import (
	"fmt"
	"log/slog"

	"inst_parser/internal/constants"
	"inst_parser/internal/models"
)

type Usecase struct {
	logger                          *slog.Logger
	accountUrlsProvider             AccountUrlsProvider
	vkGroupIDProvider               VKGroupIDProvider
	trackerService                  TrackerService
	vkClipInfoProvider              VKClipInfoProvider
	dataInserter                    DataInserter
	instagramGetReelsInfoForAccount InstagramGetReelsInfoForAccount
	youtubeChannelShortsData        YoutubeChannelShortsData
}

func NewUsecase(
	log *slog.Logger,
	accountUrlsProvider AccountUrlsProvider,
	vkGroupIDProvider VKGroupIDProvider,
	trackerService TrackerService,
	vkClipInfoProvider VKClipInfoProvider,
	dataInserter DataInserter,
	instagramGetReelsInfoForAccount InstagramGetReelsInfoForAccount,
	youtubeChannelShortsData YoutubeChannelShortsData,
) *Usecase {
	return &Usecase{
		logger:                          log,
		accountUrlsProvider:             accountUrlsProvider,
		vkGroupIDProvider:               vkGroupIDProvider,
		trackerService:                  trackerService,
		vkClipInfoProvider:              vkClipInfoProvider,
		dataInserter:                    dataInserter,
		instagramGetReelsInfoForAccount: instagramGetReelsInfoForAccount,
		youtubeChannelShortsData:        youtubeChannelShortsData,
	}
}

type (
	AccountUrlsProvider interface {
		AccountUrls(
			isSelected bool,
			sheetName, spreadsheetID string,
		) ([]*models.UrlInfo, error)
	}

	VKGroupIDProvider interface {
		GroupID(groupName string) (string, error)
	}

	VKClipInfoProvider interface {
		GetVKClipsInfoForGroup(info *models.AccountInfo) ([]*models.VKClipInfo, error)
	}

	InstagramGetReelsInfoForAccount interface {
		GetInstagramReelsInfoForAccount(info *models.AccountInfo) ([]*models.InstagramReelInfo, error)
	}

	YoutubeChannelShortsData interface {
		GetChannelIDByUsername(username string) (string, error)
		GetShortsInfoByAccountName(accountInfo *models.AccountInfo) ([]*models.YoutubeShortInfoApiResponse, error)
	}

	TrackerService interface {
		EnsureProgressSheet(spreadsheetID string) error
		StartParsing(spreadsheetID string, totalURLs int) (int, error)
		UpdateProgress(spreadsheetID string, row, progress int) error
		FinishParsing(spreadsheetID string, row int) error
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

const (
	maxCount     = 10000
	defaultCount = 12
)

func (u *Usecase) ParseAccount(
	isSelected bool,
	sheetName, spreadsheetID string,
) {
	u.logger.Info("ParsingAccount request started",
		slog.String("spreadsheet_id", spreadsheetID),
		slog.String("sheet_name", sheetName),
	)
	defer u.logger.Info("ParsingAccount request finished",
		slog.String("spreadsheet_id", spreadsheetID),
		slog.String("sheet_name", sheetName),
	)

	if err := u.trackerService.EnsureProgressSheet(spreadsheetID); err != nil {
		u.logger.Error("Failed to ensure progress sheet",
			slog.String("spreadsheet_id", spreadsheetID),
		)
	}

	accountUrls, err := u.accountUrlsProvider.AccountUrls(isSelected, sheetName, spreadsheetID)
	if err != nil {
		u.logger.Error("Failed to find account urls",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", sheetName),
			slog.String("err", err.Error()),
		)

		return
	}

	if len(accountUrls) == 0 {
		u.logger.Warn("Not found urls for parsing",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", sheetName),
		)

		return
	}

	u.logger.Info("Find groups urls successfully",
		slog.Int("count", len(accountUrls)),
		slog.String("spreadsheet_id", spreadsheetID),
		slog.String("sheet_name", sheetName),
	)

	progressRow, errStartParsing := u.trackerService.StartParsing(spreadsheetID, len(accountUrls))
	if errStartParsing != nil {
		u.logger.Error("Error starting progress tracking",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("err", errStartParsing.Error()),
		)
	}
	defer func() {
		if err = u.trackerService.FinishParsing(spreadsheetID, progressRow); err != nil {
			u.logger.Error("Error finishing progress tracking",
				slog.String("spreadsheet_id", spreadsheetID),
				slog.String("err", errStartParsing.Error()),
			)
		}
	}()

	var processedCount int
	for _, accountUrl := range accountUrls {
		accountName, parsingType, err := models.ParseSocialAccountURL(accountUrl.URL)
		if err != nil {
			u.logger.Error("Failed to parse group url",
				slog.String("url", accountUrl.URL),
				slog.String("err", err.Error()),
			)

			return
		}

		switch parsingType {
		case models.VKParsingType:
			result, processVkGroupErr := u.processVKGroup(
				accountName,
				accountUrl,
			)
			if processVkGroupErr != nil {
				u.logger.Error("Failed to get clips info",
					slog.String("spreadsheet_id", spreadsheetID),
					slog.String("sheet_name", sheetName),
					slog.String("accountName", accountName),
					slog.String("err", processVkGroupErr.Error()),
				)
				continue
			}

			if insertErr := u.dataInserter.InsertData(
				spreadsheetID,
				constants.AccountTable,
				"A:I",
				models.VKClipsInfoToInterface(result),
			); insertErr != nil {
				u.logger.Error("Failed to insert groups data", slog.String("err", insertErr.Error()))
			}
		case models.InstagramParsingType:
			result, processInstagramReelErr := u.processInstagramAccount(
				accountName,
				accountUrl,
			)
			if processInstagramReelErr != nil {
				u.logger.Error("Failed to get reel info",
					slog.String("spreadsheet_id", spreadsheetID),
					slog.String("sheet_name", sheetName),
					slog.String("accountName", accountName),
					slog.String("err", processInstagramReelErr.Error()),
				)
				continue
			}

			if insertErr := u.dataInserter.InsertData(
				spreadsheetID,
				constants.AccountTable,
				"A:I",
				models.InstagramReelInfoToInterface(result),
			); insertErr != nil {
				u.logger.Error("Failed to insert groups data", slog.String("err", insertErr.Error()))
			}

		case models.YoutubeParsingType:
			result, processYoutubeErr := u.processYoutubeAccount(
				accountName,
				accountUrl,
			)
			if processYoutubeErr != nil {
				u.logger.Error("Failed to get shoers info",
					slog.String("spreadsheet_id", spreadsheetID),
					slog.String("sheet_name", sheetName),
					slog.String("accountName", accountName),
					slog.String("err", processYoutubeErr.Error()),
				)

				continue
			}

			if insertErr := u.dataInserter.InsertData(
				spreadsheetID,
				constants.AccountTable,
				"A:I",
				models.YoutubeShortInfoApiResponseToInterface(result, accountUrl.URL),
			); insertErr != nil {
				u.logger.Error("Failed to insert groups data", slog.String("err", insertErr.Error()))
			}
		}

		processedCount++
		if updateProgressErr := u.trackerService.UpdateProgress(spreadsheetID, progressRow, processedCount); updateProgressErr != nil {
			u.logger.Error("Error updating progress", err)
		}
	}
}

func (u *Usecase) ClipMoneyParseAccount(
	accountUrl string,
) ([]*models.ClipMoneyResultRow, error) {
	u.logger.Info("ClipMoneyParseAccount request started",
		slog.String("account_url", accountUrl),
	)
	defer u.logger.Info("ClipMoneyParseAccount request finished",
		slog.String("account_url", accountUrl),
	)

	accountName, parsingType, err := models.ParseSocialAccountURL(accountUrl)
	if err != nil {
		u.logger.Error("Failed to parse group url",
			slog.String("url", accountUrl),
			slog.String("err", err.Error()),
		)

		return nil, err
	}

	switch parsingType {
	case models.VKParsingType:
		result, processVkGroupErr := u.processVKGroup(
			accountName,
			models.DefaultUrlInfo(accountUrl),
		)
		if processVkGroupErr != nil {
			u.logger.Error("Failed to get clips info",
				slog.String("accountName", accountName),
				slog.String("err", processVkGroupErr.Error()),
			)
		}

		return models.ClipMoneyResultRowFromVkClipInfo(result, accountUrl), nil
	case models.InstagramParsingType:
		result, processInstagramReelErr := u.processInstagramAccount(
			accountName,
			models.DefaultUrlInfo(accountUrl),
		)
		if processInstagramReelErr != nil {
			u.logger.Error("Failed to get reel info",
				slog.String("accountName", accountName),
				slog.String("err", processInstagramReelErr.Error()),
			)
		}

		return models.ClipMoneyResultRowFromInstagramReelInfo(result, accountUrl), nil

	case models.YoutubeParsingType:
		result, processYoutubeErr := u.processYoutubeAccount(
			accountName,
			models.DefaultUrlInfo(accountUrl),
		)
		if processYoutubeErr != nil {
			u.logger.Error("Failed to get shorts info",
				slog.String("accountName", accountName),
				slog.String("err", processYoutubeErr.Error()),
			)
		}

		return models.ClipMoneyResultRowFromYoutubeShortInfoApiResponse(result, accountUrl), nil
	}

	return nil, fmt.Errorf("unknown parsingType: %s", parsingType)
}

func (u *Usecase) processVKGroup(
	accountName string,
	accountUrl *models.UrlInfo,
) ([]*models.VKClipInfo, error) {
	groupID, groupIDErr := u.vkGroupIDProvider.GroupID(accountName)
	if groupIDErr != nil {
		u.logger.Error("Failed to get group id",
			slog.String("account_name", accountName),
			slog.String("err", groupIDErr.Error()),
		)
	}

	return u.vkClipInfoProvider.GetVKClipsInfoForGroup(
		getAccountInfo(groupID, models.VKParsingType, accountUrl),
	)
}

func (u *Usecase) processInstagramAccount(
	accountName string,
	accountUrl *models.UrlInfo,
) ([]*models.InstagramReelInfo, error) {
	return u.instagramGetReelsInfoForAccount.GetInstagramReelsInfoForAccount(
		getAccountInfo(accountName, models.InstagramParsingType, accountUrl),
	)
}

func (u *Usecase) processYoutubeAccount(
	accountName string,
	accountUrl *models.UrlInfo,
) ([]*models.YoutubeShortInfoApiResponse, error) {
	chanelID, err := u.youtubeChannelShortsData.GetChannelIDByUsername(accountName)
	if err != nil {
		u.logger.Error("Failed to get chanel id by username",
			slog.String("account_name", accountName),
			slog.String("err", err.Error()),
		)
	}
	accountInfo := getAccountInfo(accountName, models.YoutubeParsingType, accountUrl)
	accountInfo.Identification = chanelID

	return u.youtubeChannelShortsData.GetShortsInfoByAccountName(accountInfo)
}

func getAccountInfo(
	accountName string,
	parsingType models.ParsingType,
	accountUrl *models.UrlInfo,
) *models.AccountInfo {
	return &models.AccountInfo{
		Identification: accountName,
		ParsingType:    parsingType,
		AccountUrl:     accountUrl.URL,
		Count:          getCount(accountUrl.Count),
	}
}

func getCount(count int) int {
	if count <= 0 {
		return defaultCount
	}

	if count > maxCount {
		return maxCount
	}

	return count
}
