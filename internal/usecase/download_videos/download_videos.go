package download_videos

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"inst_parser/internal/models"
)

type (
	VideoDownloader interface {
		DownloadVideo(name, url, dir string) (string, error)
	}

	VkClipInfoProvider interface {
		ClipInfo(ownerID, clipID int) (*models.VKClipInfo, error)
	}
)

type Usecase struct {
	logger             *slog.Logger
	videoDownloader    VideoDownloader
	vkClipInfoProvider VkClipInfoProvider
}

func NewUsecase(
	logger *slog.Logger,
	videoDownloader VideoDownloader,
	vkClipInfoProvider VkClipInfoProvider,
) *Usecase {
	return &Usecase{
		logger:             logger,
		videoDownloader:    videoDownloader,
		vkClipInfoProvider: vkClipInfoProvider,
	}
}

type result struct {
	path string
	err  error
}

func (u *Usecase) DownloadVideos(urls []string) ([]byte, []string, error) {
	// step 1. check type of urls
	// Создаём временную директорию
	tmpDir, err := os.MkdirTemp("", "videos_*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tmp dir, err: %v", err)
	}
	defer os.RemoveAll(tmpDir) // Чистим за собой после ответа

	var (
		downloadedFiles, errors []string
		wg                      sync.WaitGroup
	)
	results := make([]result, len(urls))

	for i, url := range urls {
		_, parsingType, err := models.ParseSocialAccountURL(url)
		if err != nil {
			u.logger.Error("Failed to parse url",
				slog.String("url", url),
				slog.String("err", err.Error()),
			)

			continue
		}

		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()

			path, err := u.processOneUrl(url, tmpDir, parsingType)

			results[i] = result{path, err}
		}(i, url)
	}

	wg.Wait()

	for i, res := range results {
		if res.err != nil {
			errors = append(errors, fmt.Sprintf("URL[%d]: %v", i, res.err))
		} else {
			downloadedFiles = append(downloadedFiles, res.path)
		}
	}

	if len(downloadedFiles) == 0 {
		return nil, nil, fmt.Errorf("failed to download any videos: %v", errors)
	}

	// Создаём zip-архив во временной директории
	zipPath := filepath.Join(tmpDir, "videos.zip")
	if err := createZip(downloadedFiles, zipPath); err != nil {
		return nil, nil, fmt.Errorf("error creating archive: %v", err)
	}

	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading archive: %v", err)
	}

	return zipBytes, errors, nil
}

func (u *Usecase) processOneUrl(url, dir string, parsingType models.ParsingType) (string, error) {
	var path string

	switch parsingType {
	case models.VKParsingType:
		ownerID, clipID, err := models.ParseVkClipURL(url)
		if err != nil {
			u.logger.Error("Error parsing vk clip url",
				slog.String("url", url),
				slog.String("err", err.Error()),
			)

			return "", fmt.Errorf("error parsing vk clip url, err: %v", err)
		}

		clipInfo, err := u.vkClipInfoProvider.ClipInfo(ownerID, clipID)
		if err != nil {
			u.logger.Error("Error getting clip info",
				slog.String("url", url),
				slog.String("err", err.Error()),
			)

			return "", fmt.Errorf("error getting clip info, err: %v", err)
		}

		path, err = u.videoDownloader.DownloadVideo(strconv.Itoa(clipID), clipInfo.DownloadURL, dir)
		if err != nil {
			u.logger.Error("Error downloading video",
				slog.String("url", url),
				slog.String("err", err.Error()),
			)

			return "", fmt.Errorf("error downloading video, err: %v", err)
		}
	}

	return path, nil
}

func createZip(files []string, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("ошибка создания архива: %w", err)
	}
	defer zipFile.Close()

	writer := zip.NewWriter(zipFile)
	defer writer.Close()

	for _, filePath := range files {
		if err := addFileToZip(writer, filePath); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(writer *zip.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла %s: %w", filePath, err)
	}
	defer file.Close()

	zipEntry, err := writer.Create(filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("ошибка добавления в архив: %w", err)
	}

	_, err = io.Copy(zipEntry, file)
	return err
}
