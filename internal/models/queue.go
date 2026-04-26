package models

type QueueRequest struct {
	SpreadsheetID string
	SheetName     string
	IsSelected    bool
	Type          int // 0 = urls, 1 = account
}
