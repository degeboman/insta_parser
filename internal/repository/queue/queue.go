package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"inst_parser/internal/models"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
)

const (
	TypeURLs    = 0
	TypeAccount = 1

	streamKey     = "queue:requests"
	consumerGroup = "queue:workers"
	consumerName  = "worker-1"

	// blockDuration — время блокирующего ожидания новых сообщений.
	// Короткий таймаут позволяет корректно реагировать на отмену контекста.
	blockDuration = 2 * time.Second
)

// Queue управляет Valkey-stream-очередью.
type Queue struct {
	client valkey.Client
	logger *slog.Logger
}

// New создаёт новый экземпляр Queue и инициализирует consumer group.
// Если группа уже существует — ошибка игнорируется.
func New(client valkey.Client, logger *slog.Logger) (*Queue, error) {
	if logger == nil {
		logger = slog.Default()
	}

	q := &Queue{
		client: client,
		logger: logger,
	}

	if err := q.ensureConsumerGroup(context.Background()); err != nil {
		return nil, fmt.Errorf("queue: ensure consumer group: %w", err)
	}

	return q, nil
}

// Push сериализует запрос и публикует его в Valkey stream.
func (q *Queue) Push(ctx context.Context, req models.QueueRequest) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("queue: marshal request: %w", err)
	}

	cmd := q.client.B().Xadd().
		Key(streamKey).
		Id("*"). // авто-генерация ID
		FieldValue().
		FieldValue("payload", string(payload)).
		Build()

	if err := q.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("queue: xadd: %w", err)
	}

	q.logger.Debug("queue: pushed request",
		"spreadsheet_id", req.SpreadsheetID,
		"sheet_name", req.SheetName,
		"type", req.Type,
	)

	return nil
}

// Watcher читает сообщения из stream по одному, вызывает нужный обработчик
// и подтверждает (ACK) + удаляет обработанное сообщение.
// Блокируется до отмены ctx.
func (q *Queue) Watcher(
	ctx context.Context,
	executeUrls func(isSelected bool, spreadsheetID, sheetName string),
	executeAccount func(isSelected bool, spreadsheetID, sheetName string),
) {
	q.logger.Info("queue: watcher started")
	defer q.logger.Info("queue: watcher stopped")

	// Сначала обрабатываем pending-сообщения (упавшие в прошлый раз).
	q.recoverPending(ctx, executeUrls, executeAccount)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgID, req, err := q.readOne(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			// Временная ошибка сети / таймаут — ждём и пробуем снова.
			q.logger.Warn("queue: read error, retrying", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		if msgID == "" {
			// Нет новых сообщений (истёк block-таймаут) — повторяем цикл.
			continue
		}

		q.dispatch(req, executeUrls, executeAccount)

		if err := q.ackAndDelete(ctx, msgID); err != nil {
			q.logger.Error("queue: ack/delete failed",
				"message_id", msgID,
				"error", err,
			)
			// Не прерываем работу — следующий запуск подберёт pending.
		}
	}
}

// ── internal ──────────────────────────────────────────────────────────────────

// ensureConsumerGroup создаёт группу MKSTREAM (stream создаётся при необходимости).
func (q *Queue) ensureConsumerGroup(ctx context.Context) error {
	cmd := q.client.B().XgroupCreate().
		Key(streamKey).
		Group(consumerGroup).
		Id("0"). // читать всё с начала
		Mkstream().
		Build()

	err := q.client.Do(ctx, cmd).Error()
	if err != nil && !isBusyGroupError(err) {
		return err
	}

	return nil
}

// readOne выполняет XREADGROUP BLOCK … COUNT 1 и возвращает первое сообщение.
// Возвращает ("", nil, nil) если истёк blockDuration и сообщений нет.
func (q *Queue) readOne(ctx context.Context) (msgID string, req models.QueueRequest, err error) {
	cmd := q.client.B().Xreadgroup().
		Group(consumerGroup, consumerName).
		Count(1).
		Block(blockDuration.Milliseconds()).
		Streams().
		Key(streamKey).
		Id(">"). // только новые, ещё не доставленные
		Build()

	result, err := q.client.Do(ctx, cmd).AsXRead()
	if err != nil {
		// valkey-go возвращает ошибку "redis: nil" при пустом ответе на BLOCK.
		if isNilError(err) {
			return "", models.QueueRequest{}, nil
		}
		return "", models.QueueRequest{}, fmt.Errorf("xreadgroup: %w", err)
	}

	entries, ok := result[streamKey]
	if !ok || len(entries) == 0 {
		return "", models.QueueRequest{}, nil
	}

	entry := entries[0]
	msgID = entry.ID

	payloadStr, ok := entry.FieldValues["payload"]
	if !ok {
		return msgID, models.QueueRequest{}, fmt.Errorf("queue: missing payload field in message %s", msgID)
	}

	if err := json.Unmarshal([]byte(payloadStr), &req); err != nil {
		return msgID, models.QueueRequest{}, fmt.Errorf("queue: unmarshal payload: %w", err)
	}

	return msgID, req, nil
}

// dispatch вызывает нужный обработчик в зависимости от типа запроса.
func (q *Queue) dispatch(
	req models.QueueRequest,
	executeUrls func(bool, string, string),
	executeAccount func(bool, string, string),
) {
	q.logger.Info("queue: dispatching request",
		"spreadsheet_id", req.SpreadsheetID,
		"sheet_name", req.SheetName,
		"type", req.Type,
		"is_selected", req.IsSelected,
	)

	switch req.Type {
	case TypeURLs:
		executeUrls(req.IsSelected, req.SheetName, req.SpreadsheetID)
	case TypeAccount:
		executeAccount(req.IsSelected, req.SheetName, req.SpreadsheetID)
	default:
		q.logger.Warn("queue: unknown request type", "type", req.Type)
	}
}

// ackAndDelete подтверждает обработку сообщения и удаляет его из stream.
func (q *Queue) ackAndDelete(ctx context.Context, msgID string) error {
	// XACK — помечаем как обработанное в группе.
	ackCmd := q.client.B().Xack().
		Key(streamKey).
		Group(consumerGroup).
		Id(msgID).
		Build()

	if err := q.client.Do(ctx, ackCmd).Error(); err != nil {
		return fmt.Errorf("xack %s: %w", msgID, err)
	}

	// XDEL — удаляем из самого stream, чтобы он не рос бесконечно.
	delCmd := q.client.B().Xdel().
		Key(streamKey).
		Id(msgID).
		Build()

	if err := q.client.Do(ctx, delCmd).Error(); err != nil {
		return fmt.Errorf("xdel %s: %w", msgID, err)
	}

	q.logger.Debug("queue: message ack+deleted", "message_id", msgID)
	return nil
}

// recoverPending обрабатывает сообщения, доставленные ранее, но не ACK-нутые
// (например, после перезапуска сервиса).
func (q *Queue) recoverPending(
	ctx context.Context,
	executeUrls func(bool, string, string),
	executeAccount func(bool, string, string),
) {
	q.logger.Info("queue: recovering pending messages")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		cmd := q.client.B().Xreadgroup().
			Group(consumerGroup, consumerName).
			Count(1).
			Streams().
			Key(streamKey).
			Id("0"). // читаем pending данного consumer-а
			Build()

		result, err := q.client.Do(ctx, cmd).AsXRead()
		if err != nil {
			if isNilError(err) {
				break
			}
			q.logger.Error("queue: recover pending read error", "error", err)
			return
		}

		entries, ok := result[streamKey]
		if !ok || len(entries) == 0 {
			break
		}

		entry := entries[0]

		payloadStr, ok := entry.FieldValues["payload"]
		if !ok {
			q.logger.Warn("queue: pending message missing payload, deleting", "id", entry.ID)
			_ = q.ackAndDelete(ctx, entry.ID)
			continue
		}

		var req models.QueueRequest
		if err := json.Unmarshal([]byte(payloadStr), &req); err != nil {
			q.logger.Warn("queue: pending message unmarshal failed, deleting",
				"id", entry.ID, "error", err)
			_ = q.ackAndDelete(ctx, entry.ID)
			continue
		}

		q.dispatch(req, executeUrls, executeAccount)

		if err := q.ackAndDelete(ctx, entry.ID); err != nil {
			q.logger.Error("queue: recover ack/delete failed", "id", entry.ID, "error", err)
			return
		}
	}

	q.logger.Info("queue: pending messages recovered")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func isBusyGroupError(err error) bool {
	if err == nil {
		return false
	}
	// Valkey/Redis возвращает "BUSYGROUP Consumer Group name already exists"
	var vkErr *valkey.ValkeyError
	if errors.As(err, &vkErr) {
		return vkErr.IsBusyGroup()
	}
	return false
}

func isNilError(err error) bool {
	if err == nil {
		return false
	}
	var vkErr *valkey.ValkeyError
	if errors.As(err, &vkErr) {
		return vkErr.IsNil()
	}
	return false
}
