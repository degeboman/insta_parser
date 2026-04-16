package utils

import (
	"time"

	"inst_parser/internal/constants"
)

var moscow, _ = time.LoadLocation("Europe/Moscow")

var validDateFormats = []string{
	time.RFC3339,
	constants.YoutubeParsingDateFormat,
	constants.ParsingDateFormat,
}

func ParsingDate() string {
	return time.Now().In(moscow).Format(time.DateTime)
}

func FormatParsingDate(date string) string {
	for _, format := range validDateFormats {
		t, err := time.Parse(format, date)
		if err != nil {
			continue
		}

		return t.Format(time.DateTime)
	}

	return ""
}

func PublishDate(pubTime time.Time) string {
	return pubTime.In(moscow).Format(time.DateTime)
}
