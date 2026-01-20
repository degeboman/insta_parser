package models

import "strings"

type ParsingType string

const (
	Instagram ParsingType = "instagram"
	VK        ParsingType = "vk"
	Unknown   ParsingType = "unknown"
)

var parsingTypes = map[ParsingType]struct{}{
	Instagram: struct{}{},
	VK:        struct{}{},
}

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
		return VK
	}

	if strings.Contains(urlLower, "vk.ru") {
		return VK
	}

	// Проверяем, содержит ли URL домен instagram.com
	if strings.Contains(urlLower, "instagram.com") {
		return Instagram
	}

	return Unknown
}
