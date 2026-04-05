package utils

import (
	"time"

	"inst_parser/internal/constants"
)

var moscow, _ = time.LoadLocation("Europe/Moscow")

func ParsingDate() string {
	return time.Now().In(moscow).Format(constants.ParsingDateFormat)
}

func PublishDate(pubTime time.Time) string {
	return pubTime.In(moscow).Format(constants.ParsingDateFormat)
}
