package main

import (
	"inst_parser/internal/google_sheet"
	"inst_parser/internal/rapid"
	"log"
	"net/http"

	"inst_parser/internal/config"
	"inst_parser/internal/handler"
	"inst_parser/internal/logger"
)

func main() {
	cfg := config.MustLoad()
	l := logger.NewLogger()

	l.Info("Starting server")

	rapidSrv := rapid.NewService(cfg.Rapid.ApiKey, l)
	sheetSrv := google_sheet.NewService(cfg.GoogleDriveCredentials)

	parsingHandler := handler.NewParsingHandler(l, rapidSrv, sheetSrv)

	http.HandleFunc("/parsing", parsingHandler.Parsing)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
