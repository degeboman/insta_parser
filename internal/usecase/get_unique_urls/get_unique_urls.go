package get_unique_urls

import (
	"context"
	"inst_parser/internal/models"
	"log/slog"
)

type (
	DataInserter interface {
		InsertData(
			spreadsheetID,
			sheetName,
			rangeData string,
			data [][]interface{},
		) error

		InsertDataV2(
			spreadsheetID,
			sheetName,
			rangeData string,
			data [][]interface{},
		) error

		InsertDataWithClear(
			spreadsheetID,
			sheetName,
			rangeData string,
			data [][]interface{},
		) error
	}

	UrlsProvider interface {
		GetAll(ctx context.Context, spreadsheetID string) ([]*models.ResultRowUrl, error)
	}
)

type Usecase struct {
	l            *slog.Logger
	dataInserter DataInserter
	urlsProvider UrlsProvider
}

func NewGetUrlsUsecase(
	l *slog.Logger,
	dataInserter DataInserter,
	urlsProvider UrlsProvider,
) *Usecase {
	return &Usecase{
		l:            l,
		dataInserter: dataInserter,
		urlsProvider: urlsProvider,
	}
}

func (u *Usecase) GetUniqueUrls(spreadsheetID string) error {
	urls, err := u.urlsProvider.GetAll(context.Background(), spreadsheetID)
	if err != nil {
		return err
	}

	if err = u.dataInserter.InsertDataWithClear(
		spreadsheetID,
		"Уникальные ссылки",
		"A:I",
		models.ResultRowsUniqueToInterface(urls),
	); err != nil {
		u.l.Error("ParsingUrls URLs returned an error",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("sheet_name", "Уникальные ссылки"),
			slog.String("err", err.Error()),
		)
		return err
	}

	return nil
}
