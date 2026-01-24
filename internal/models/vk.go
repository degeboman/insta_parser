package models

import (
	"log"
	"time"
)

type ClipInfo struct {
	OwnerID     int
	GroupUrl    string
	URL         string
	ClipID      int
	Views       int
	Likes       int
	Comments    int
	Shares      int
	Description string
	ER          string
	Virality    string
	ParsingDate string
	PublishDate string
	Date        time.Time
}

func EmptyClipInfo(url string) *ClipInfo {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	parsingDate := time.Now().In(moscow).Format("02.01.2006 15:04")
	return &ClipInfo{
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

type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Clips  []APIVKClip `json:"clips"`
		Cursor string      `json:"cursor"`
	} `json:"data"`
}

type APIVKClip struct {
	ID          int    `json:"id"`
	OwnerID     int    `json:"owner_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Duration    int    `json:"duration"`
	Views       int    `json:"views"`
	LocalViews  int    `json:"local_views"`
	Date        int    `json:"date"`
	Reposts     struct {
		Count int `json:"count"`
	} `json:"reposts"`
	Likes struct {
		Count int `json:"count"`
	} `json:"likes"`
	Comments int `json:"comments"`
}

type GroupInfoPair struct {
	OwnerID  string
	GroupUrl string
	Count    int
}
