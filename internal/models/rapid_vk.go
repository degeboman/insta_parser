package models

type RapidVkScraperUserClipsResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Clips  []RapidVkScraperClip `json:"clips"`
		Cursor string               `json:"cursor"`
	} `json:"data"`
}
type RapidVkScraperClip struct {
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
