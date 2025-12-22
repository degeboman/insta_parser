package models

import (
	"log"
	"time"
)

type ResultRow struct {
	URL         string
	Views       int64
	Likes       int64
	Comments    int64
	Shares      int64
	ER          string
	Virality    string
	ParsingDate string
	PublishDate string
}

func EmptyResultRow(url string) *ResultRow {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	parsingDate := time.Now().In(moscow).Format("02.01.2006 15:04")
	return &ResultRow{
		URL:         url,
		Views:       0,
		Likes:       0,
		Comments:    0,
		Shares:      0,
		ER:          "0",
		Virality:    "0",
		ParsingDate: parsingDate,
		PublishDate: "unknown",
	}
}
