package utils

import "time"

var moscow, _ = time.LoadLocation("Europe/Moscow")

func ParsingDate() string {
	return time.Now().In(moscow).Format("02.01.2006 15:04")
}
