package main

import (
	"inst_parser/internal/repository/video_downloader"
	"log"
	"net/http"

	_ "inst_parser/docs"
	"inst_parser/internal/config"
	"inst_parser/internal/constants"
	"inst_parser/internal/handlers"
	"inst_parser/internal/logger"
	"inst_parser/internal/repository/google_sheet"
	"inst_parser/internal/repository/progress"
	"inst_parser/internal/repository/rapid"
	"inst_parser/internal/repository/vk"
	"inst_parser/internal/repository/youtube"
	"inst_parser/internal/usecase/download_videos"
	"inst_parser/internal/usecase/parsing_account"
	"inst_parser/internal/usecase/parsing_urls"
	"inst_parser/internal/usecase/search_url"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Parser social media videos
// @version         1.0
// @description     This is a sample server for parsing social media videos.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@your-domain.com

// @host     hammerhead-app-xw9wl.ondigitalocean.app
// @BasePath  /

func main() {
	cfg := config.MustLoad()
	l := logger.NewLogger()

	l.Info("Starting server")

	googleSheetRepo := google_sheet.NewRepository(cfg.GoogleDriveCredentials)
	progressSrv := progress.NewProgressTracker(googleSheetRepo.SheetsService)
	urlSrv := search_url.NewUrlsService(l, googleSheetRepo.SheetsService)
	vkRepo := vk.NewRepository(l, cfg.VK.Token)
	rapidRepo := rapid.NewRepository(cfg.Rapid.ApiKey, l, vkRepo)
	youtubeRepo := youtube.NewYouTubeClient(l, cfg.Youtube.YoutubeToken)
	videoDownloaderRepo := video_downloader.NewRepository()

	parsingUrlsUsecase := parsing_urls.NewUsecase(
		l,
		urlSrv,
		googleSheetRepo,
		rapidRepo,
		vkRepo,
		progressSrv,
		youtubeRepo,
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
		youtubeRepo,
		rapidRepo,
	)

	downloadVideosUsecase := download_videos.NewUsecase(l, videoDownloaderRepo, vkRepo, rapidRepo)

	parsingUrlsHandler := handlers.NewParsingUrlsHandler(l, parsingUrlsUsecase)
	clipMoneyParsingUrlHandler := handlers.NewClipMoneyParsingUrl(l, parsingUrlsUsecase)
	parsingAccountHandler := handlers.NewParsingAccountsHandler(l, parsingAccountUsecase)
	clipMoneyParsingAccountHandler := handlers.NewClipMoneyParsingAccount(l, parsingAccountUsecase)
	downloadVideosHandler := handlers.NewDownloadVideos(l, downloadVideosUsecase)

	http.HandleFunc(constants.ParsingUrls, parsingUrlsHandler.ParsingUrls)
	http.HandleFunc(constants.ParsingAccount, parsingAccountHandler.ParsingAccount)
	http.HandleFunc(constants.ClipMoneyParsingAccount, clipMoneyParsingAccountHandler.ClipMoneyParsingAccount)
	http.HandleFunc(constants.ClipMoneyParsingUrl, clipMoneyParsingUrlHandler.ClipMoneyParsingUrl)
	http.HandleFunc(constants.DownloadVideos, downloadVideosHandler.DownloadVideos)
	http.HandleFunc(constants.DownloadVideosGet, downloadVideosHandler.DownloadVideosGet)

	http.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // URL для вашей swagger документации
	))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
