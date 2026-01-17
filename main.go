package main

import (
	"log"
	"net/http"

	"inst_parser/internal/config"
	"inst_parser/internal/google_sheet"
	"inst_parser/internal/handler"
	"inst_parser/internal/logger"
	"inst_parser/internal/rapid"
	"inst_parser/internal/vk"
)

func main() {
	cfg := config.MustLoad()
	l := logger.NewLogger()

	l.Info("Starting server")

	vksrv := vk.NewVKClipService(cfg.VK.Token)
	sheetSrv := google_sheet.NewService(cfg.GoogleDriveCredentials)
	rapidSrv := rapid.NewService(cfg.Rapid.ApiKey, l, sheetSrv, vksrv)
	urlSrv := google_sheet.NewUrlsService(l, sheetSrv.SheetsService)

	parsingHandler := handler.NewParsingHandler(l, rapidSrv, sheetSrv, urlSrv)

	http.HandleFunc("/parsing2", parsingHandler.Parsing2)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
