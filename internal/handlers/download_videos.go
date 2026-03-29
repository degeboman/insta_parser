package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"inst_parser/internal/usecase/download_videos"
)

type DownloadVideos struct {
	logger  *slog.Logger
	usecase *download_videos.Usecase
}

func NewDownloadVideos(
	logger *slog.Logger,
	usecase *download_videos.Usecase,
) *DownloadVideos {
	return &DownloadVideos{
		logger:  logger,
		usecase: usecase,
	}
}

type (
	// DownloadVideosRequest
	DownloadVideosRequest struct {
		Urls []string `json:"urls" example:"['https://vk.com/clip-226676596_456242668']"` // Videos URL to download
	}

	// DownloadVideosResponse
	DownloadVideosResponse struct {
		Success bool   `json:"success" example:"true"`                    // Operation success status
		Message string `json:"message" example:"URL parsed successfully"` // Response message
	}
)

// DownloadVideos godoc
// @Summary      Download video by URL
// @Description  Download video vk
// @Tags         download
// @Accept       json
// @Produce      json
// @Param        request body DownloadVideosRequest true "URL to parse"
// @Success      200  {object}  DownloadVideosResponse  "Successfully parsed URL"
// @Failure      400  {object}  DownloadVideosResponse  "Invalid request format or missing URL"
// @Failure      405  {object}  DownloadVideosResponse  "Method not allowed"
// @Failure      500  {object}  DownloadVideosResponse  "Internal server error"
// @Router       /download_videos [post]
func (h *DownloadVideos) DownloadVideos(w http.ResponseWriter, r *http.Request) {
	const maxUrlsCount = 50
	// Разрешаем только POST метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON из тела запроса
	var req DownloadVideosRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp := DownloadVideosResponse{
			Success: false,
			Message: "Invalid JSON format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(req.Urls) == 0 {
		resp := DownloadVideosResponse{
			Success: false,
			Message: "url is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(req.Urls) > maxUrlsCount {
		resp := DownloadVideosResponse{
			Success: false,
			Message: "url is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	zipBytes, errors, err := h.usecase.DownloadVideos(req.Urls)
	if err != nil {
		h.logger.Error("Failed to download videos",
			slog.String("err", err.Error()),
		)

		resp := DownloadVideosResponse{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=videos.zip")
	w.Header().Set("Content-Length", strconv.Itoa(len(zipBytes)))

	if len(errors) > 0 {
		w.Header().Set("X-Download-Errors", strings.Join(errors, "; "))
	}

	w.Write(zipBytes)
}

// DownloadVideosGet godoc
// @Summary      Download video by URL
// @Description  Download video vk
// @Tags         download
// @Accept       json
// @Produce      json
// @Param        request body DownloadVideosRequest true "URL to parse"
// @Success      200  {object}  DownloadVideosResponse  "Successfully parsed URL"
// @Failure      400  {object}  DownloadVideosResponse  "Invalid request format or missing URL"
// @Failure      405  {object}  DownloadVideosResponse  "Method not allowed"
// @Failure      500  {object}  DownloadVideosResponse  "Internal server error"
// @Router       /download_videos_get [get]
func (h *DownloadVideos) DownloadVideosGet(w http.ResponseWriter, r *http.Request) {
	const maxUrlsCount = 50

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	urls := r.URL.Query()["urls"]

	if len(urls) == 0 {
		resp := DownloadVideosResponse{
			Success: false,
			Message: "urls is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(urls) > maxUrlsCount {
		resp := DownloadVideosResponse{
			Success: false,
			Message: fmt.Sprintf("too many urls, max is %d", maxUrlsCount),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	zipBytes, errors, err := h.usecase.DownloadVideos(urls)
	if err != nil {
		h.logger.Error("Failed to download videos",
			slog.String("err", err.Error()),
		)

		resp := DownloadVideosResponse{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=videos.zip")
	w.Header().Set("Content-Length", strconv.Itoa(len(zipBytes)))

	if len(errors) > 0 {
		w.Header().Set("X-Download-Errors", strings.Join(errors, "; "))
	}

	w.Write(zipBytes)
}
