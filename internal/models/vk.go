package models

import "time"

type ClipInfo struct {
	OwnerID  int       `json:"owner_id"`
	ClipID   int       `json:"clip_id"`
	Views    int       `json:"views"`
	Likes    int       `json:"likes"`
	Comments int       `json:"comments"`
	Reposts  int       `json:"reposts"`
	Date     time.Time `json:"date"`
}
