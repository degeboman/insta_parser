package main

import (
	"context"
	"inst_parser/internal/repository/queue"
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
	"inst_parser/internal/repository/video_downloader"
	"inst_parser/internal/repository/vk"
	"inst_parser/internal/repository/youtube"
	"inst_parser/internal/store"
	"inst_parser/internal/usecase/download_videos"
	"inst_parser/internal/usecase/parsing_account"
	"inst_parser/internal/usecase/parsing_urls"
	"inst_parser/internal/usecase/search_url"

	"github.com/rs/cors"
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
	//client := godo.NewFromToken(token)
	//cluster, _, err := client.Databases.Get(ctx, "9cc10173-e9ea-4176-9dbc-a4cee4c4ff30")

	valkey := store.NewValKeyClient(cfg.ValKey)
	defer valkey.Close()

	googleSheetRepo := google_sheet.NewRepository(cfg.GoogleDriveCredentials)
	queueCli, err := queue.New(valkey, l)
	if err != nil {
		log.Fatal(err)
	}
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
		rapidRepo,
	)

	downloadVideosUsecase := download_videos.NewUsecase(l, videoDownloaderRepo, vkRepo, rapidRepo)

	parsingUrlsHandler := handlers.NewParsingUrlsHandler(l, queueCli, parsingUrlsUsecase)
	clipMoneyParsingUrlHandler := handlers.NewClipMoneyParsingUrl(l, parsingUrlsUsecase)
	parsingAccountHandler := handlers.NewParsingAccountsHandler(l, queueCli, parsingAccountUsecase)
	clipMoneyParsingAccountHandler := handlers.NewClipMoneyParsingAccount(l, parsingAccountUsecase)
	downloadVideosHandler := handlers.NewDownloadVideos(l, downloadVideosUsecase)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go queueCli.Watcher(
		ctx,
		parsingUrlsUsecase.ParseUrls,
		parsingAccountUsecase.ParseAccount,
	)

	mux := http.NewServeMux()

	mux.HandleFunc(constants.ParsingUrls, parsingUrlsHandler.ParsingUrls)
	mux.HandleFunc(constants.ParsingAccount, parsingAccountHandler.ParsingAccount)
	mux.HandleFunc(constants.ClipMoneyParsingAccount, clipMoneyParsingAccountHandler.ClipMoneyParsingAccount)
	mux.HandleFunc(constants.ClipMoneyParsingUrl, clipMoneyParsingUrlHandler.ClipMoneyParsingUrl)
	mux.HandleFunc(constants.DownloadVideos, downloadVideosHandler.DownloadVideos)
	mux.HandleFunc(constants.DownloadVideosGet, downloadVideosHandler.DownloadVideosGet)
	mux.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Настройка CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"POST", "GET", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Accept", "Authorization"},
		AllowCredentials: false,
	})

	// Оборачиваем роутер в CORS middleware
	handler := c.Handler(mux)

	http.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // URL для вашей swagger документации
	))

	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
