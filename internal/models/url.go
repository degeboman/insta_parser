package models

import (
	"fmt"
	"regexp"
	"strconv"
)

const (
	instagramPattern = `(?:https?://)?(?:www\.)?instagram\.com/([^/?#]+)`
	vkPattern        = `(?:https?://)?(?:www\.)?vk\.(?:com|ru)/([^/?#]+)`
	telegramPattern  = `(?:https?://)?(?:www\.)?t\.me/([^/?#]+)`
	youtubePattern   = `(?:https?://)?(?:www\.)?youtube\.com/(?:c/|channel/|@)?([^/?#]+)`
	tiktokPattern    = `(?:https?://)?(?:www\.)?tiktok\.com/@([^/?#]+)`
)

func ParseSocialAccountURL(url string) (
	account string,
	parsingType ParsingType,
	err error,
) {
	// Паттерны для различных социальных сетей
	patterns := map[ParsingType]*regexp.Regexp{
		InstagramParsingType: regexp.MustCompile(instagramPattern),
		VKParsingType:        regexp.MustCompile(vkPattern),
		TelegramParsingType:  regexp.MustCompile(telegramPattern),
		YoutubeParsingType:   regexp.MustCompile(youtubePattern),
		TiktokParsingType:    regexp.MustCompile(tiktokPattern),
	}

	for platformName, re := range patterns {
		matches := re.FindStringSubmatch(url)
		if matches != nil && len(matches) > 1 && matches[1] != "" {
			return matches[1], platformName, nil
		}
	}

	return "", "", fmt.Errorf("unsupported social media URL or invalid format")
}

// ParseClipURL извлекает owner_id и clip_id из URL
func ParseVkClipURL(url string) (int, int, error) {
	re := regexp.MustCompile(`clip(-?\d+)_(\d+)`)
	matches := re.FindStringSubmatch(url)

	if len(matches) != 3 {
		return 0, 0, fmt.Errorf("invalid clip URL format")
	}

	ownerID, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, err
	}

	clipID, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, err
	}

	return ownerID, clipID, nil
}
