package queue

import (
	"context"
	"fmt"
	"sync"

	"inst_parser/internal/models"
)

const (
	QueueSize  = 100
	MaxWorkers = 100
)

type Queue struct {
	ch        chan models.QueueRequest
	semaphore chan struct{}

	mu    sync.Mutex
	locks map[string]chan struct{} // ID → канал-блокировка
}

func NewQueue() *Queue {
	return &Queue{
		ch:        make(chan models.QueueRequest, QueueSize),
		semaphore: make(chan struct{}, MaxWorkers),
		locks:     make(map[string]chan struct{}),
	}
}

// Enqueue добавляет запрос в очередь.
// Возвращает ошибку если очередь полна.
func (q *Queue) Enqueue(req models.QueueRequest) error {
	select {
	case q.ch <- req:
		return nil
	default:
		return fmt.Errorf("queue is full (max %d)", QueueSize)
	}
}

// Watcher запускает цикл обработки очереди.
// Завершается при отмене контекста.
func (q *Queue) Watcher(
	ctx context.Context,
	executeUrls func(bool, string, string),
	executeAccount func(bool, string, string),
) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-q.ch:
			q.semaphore <- struct{}{} // захватываем слот (не более MaxWorkers)
			go func(r models.QueueRequest) {
				defer func() { <-q.semaphore }()
				q.processWithIDLock(ctx, r, executeUrls, executeAccount)
			}(req)
		}
	}
}

// processWithIDLock ждёт, если задача с тем же ID уже выполняется.
func (q *Queue) processWithIDLock(
	ctx context.Context,
	req models.QueueRequest,
	executeUrls func(bool, string, string),
	executeAccount func(bool, string, string),
) {
	for {
		q.mu.Lock()
		if _, running := q.locks[req.SpreadsheetID]; !running {
			// ID свободен — занимаем
			q.locks[req.SpreadsheetID] = make(chan struct{})
			q.mu.Unlock()
			break
		}
		// ID занят — берём канал и ждём снаружи мьютекса
		wait := q.locks[req.SpreadsheetID]
		q.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-wait:
			// предыдущая задача завершилась, пробуем снова
		}
	}

	// Выполняем задачу
	if req.Type == 0 {
		executeUrls(req.IsSelected, req.SheetName, req.SpreadsheetID)
	} else if req.Type == 1 {
		executeAccount(req.IsSelected, req.SheetName, req.SpreadsheetID)
	}

	// Освобождаем ID и уведомляем ожидающих
	q.mu.Lock()
	ch := q.locks[req.SpreadsheetID]
	delete(q.locks, req.SpreadsheetID)
	q.mu.Unlock()

	close(ch) // все ожидающие этот ID разблокируются и попробуют снова
}
