package schedule

import (
	"log/slog"
	"sync"
)

type Item struct {
}

type Service struct {
	log *slog.Logger
	mu  sync.Mutex
	//items []*
}
