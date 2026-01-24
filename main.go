package main

import (
	"log"
	"net/http"

	"inst_parser/internal/config"
	"inst_parser/internal/google_sheet"
	"inst_parser/internal/handlers"
	"inst_parser/internal/logger"
	"inst_parser/internal/rapid"
	"inst_parser/internal/vk"
)

func main() {
	cfg := config.MustLoad()
	l := logger.NewLogger()

	l.Info("Starting server")

	vksrv := vk.NewVKService(cfg.VK.Token)
	sheetSrv := google_sheet.NewService(cfg.GoogleDriveCredentials)
	tracker := google_sheet.NewProgressTracker(sheetSrv.SheetsService)
	rapidSrv := rapid.NewService(cfg.Rapid.ApiKey, l, sheetSrv, vksrv, tracker)
	urlSrv := google_sheet.NewUrlsService(l, sheetSrv.SheetsService)

	parsingUrlsHandler := handlers.NewParsingUrlsHandler(l, rapidSrv, sheetSrv, urlSrv)
	parsingVkGroupsHandler := handlers.NewParsingVkGroupsHandler(l, sheetSrv, urlSrv, vksrv, rapidSrv, tracker)

	http.HandleFunc("/parsing2", parsingUrlsHandler.ParsingUrls)
	http.HandleFunc("/parsing_vk_groups", parsingVkGroupsHandler.ParsingVkGroups)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
