package video_downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Repository struct {
}

func NewRepository() *Repository {
	return &Repository{}
}

// DownloadVideo скачивает видео по ссылке в корень проекта
func (r *Repository) DownloadVideo(name, url, dir string) (string, error) {
	fileName := filepath.Base(url)
	if idx := strings.Index(fileName, "?"); idx != -1 {
		fileName = fileName[:idx]
	}
	if fileName == "" || fileName == "." {
		fileName = name + ".mp4"
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("сервер вернул статус %s для %s", resp.Status, url)
	}

	filePath := filepath.Join(dir, fileName)
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("ошибка создания файла: %w", err)
	}
	defer outFile.Close()

	if _, err = io.Copy(outFile, resp.Body); err != nil {
		return "", fmt.Errorf("ошибка записи файла: %w", err)
	}

	return filePath, nil
}
