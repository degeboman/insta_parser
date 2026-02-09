package main

import (
	"log"
	"net/http"

	"inst_parser/internal/config"
	"inst_parser/internal/constants"
	"inst_parser/internal/handlers"
	"inst_parser/internal/logger"
	"inst_parser/internal/repository/google_sheet"
	"inst_parser/internal/repository/progress"
	"inst_parser/internal/repository/rapid"
	"inst_parser/internal/repository/vk"
	"inst_parser/internal/usecase/parsing_account"
	"inst_parser/internal/usecase/parsing_urls"
	"inst_parser/internal/usecase/search_url"
)

func main() {
	cfg := config.MustLoad()
	l := logger.NewLogger()

	l.Info("Starting server")

	googleSheetRepo := google_sheet.NewRepository(cfg.GoogleDriveCredentials)
	progressSrv := progress.NewProgressTracker(googleSheetRepo.SheetsService)
	urlSrv := search_url.NewUrlsService(l, googleSheetRepo.SheetsService)
	rapidRepo := rapid.NewRepository(cfg.Rapid.ApiKey, l)
	vkRepo := vk.NewRepository(l, cfg.VK.Token)

	parsingUrlsUsecase := parsing_urls.NewUsecase(
		l,
		urlSrv,
		googleSheetRepo,
		rapidRepo,
		vkRepo,
		progressSrv,
		rapidRepo,
	)

	parsingAccountUsecase := parsing_account.NewUsecase(
		l,
		urlSrv,
		vkRepo,
		progressSrv,
		rapidRepo,
		googleSheetRepo,
		rapidRepo,
		rapidRepo,
	)

	parsingUrlsHandler := handlers.NewParsingUrlsHandler(l, parsingUrlsUsecase)
	parsingAccountHandler := handlers.NewParsingVkGroupsHandler(l, parsingAccountUsecase)

	http.HandleFunc(constants.ParsingUrls, parsingUrlsHandler.ParsingUrls)
	http.HandleFunc(constants.ParsingAccount, parsingAccountHandler.ParsingAccount)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
