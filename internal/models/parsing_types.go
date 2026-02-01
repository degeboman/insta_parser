package models

import "strings"

type ParsingType string

const (
	InstagramParsingType ParsingType = "instagram"
	VKParsingType        ParsingType = "vk"
	YoutubeParsingType   ParsingType = "youtube"
	TiktokParsingType    ParsingType = "tiktok"
	TelegramParsingType  ParsingType = "telegram"
	UnknownParsingType   ParsingType = "unknown"
)

func IsAvailableByParsingType(url string, parsingTypes []ParsingType) bool {
	for _, parsingType := range parsingTypes {
		if strings.Contains(url, string(parsingType)) {
			return true
		}
	}

	return false
}

func ParsingTypeByUrl(url string) ParsingType {
	urlLower := strings.ToLower(url)

	// Проверяем, содержит ли URL домен vk.com
	if strings.Contains(urlLower, "vk.com") {
		return VKParsingType
	}

	if strings.Contains(urlLower, "vk.ru") {
		return VKParsingType
	}

	// Проверяем, содержит ли URL домен instagram.com
	if strings.Contains(urlLower, "instagram.com") {
		return InstagramParsingType
	}

	return UnknownParsingType
}
