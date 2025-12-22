package models

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
