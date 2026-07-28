package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"inst_parser/internal/models"
	"log/slog"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"
)

const (
	TypeURLs      = 0
	TypeAccount   = 1
	TypeURLsV2    = 2
	TypeAccountV2 = 3

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
	executeUrlsV2 func(isSelected bool, spreadsheetID, sheetName string),
	executeAccount func(isSelected bool, spreadsheetID, sheetName string),
	executeAccountV2 func(isSelected bool, spreadsheetID, sheetName string),
) {
	q.logger.Info("queue: watcher started")
	defer q.logger.Info("queue: watcher stopped")

	// Сначала обрабатываем pending-сообщения (упавшие в прошлый раз).
	q.recoverPending(ctx, executeUrls, executeUrlsV2, executeAccount, executeAccountV2)

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

		q.dispatch(req, executeUrls, executeUrlsV2, executeAccount, executeAccountV2)

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
	executeUrlsV2 func(bool, string, string),
	executeAccount func(bool, string, string),
	executeAccountV2 func(bool, string, string),
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
	case TypeAccountV2:
		executeAccountV2(req.IsSelected, req.SheetName, req.SpreadsheetID)
	case TypeURLsV2:
		executeUrlsV2(req.IsSelected, req.SheetName, req.SpreadsheetID)
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
	executeUrlsV2 func(bool, string, string),
	executeAccount func(bool, string, string),
	executeAccountV2 func(bool, string, string),
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

		q.dispatch(req, executeUrls, executeUrlsV2, executeAccount, executeAccountV2)

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

// Вспомогательные функции для генерации ключей
func dataKey(spreadsheetID string) string {
	return "video:data:" + spreadsheetID
}

func viewsKey(spreadsheetID string) string {
	return "video:views:" + spreadsheetID
}

// CreateOrUpdate – вставка или обновление одной записи
func (q *Queue) CreateOrUpdate(ctx context.Context, row *models.ResultRowUrl) error {
	data, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal row: %w", err)
	}

	// Hash: поле = URL, значение = JSON
	cmd := q.client.B().Hset().Key(dataKey(row.SpreadsheetID)).FieldValue().FieldValue(row.URL, string(data)).Build()
	if err := q.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("hset: %w", err)
	}

	// Sorted Set: member = URL, score = Views
	zCmd := q.client.B().Zadd().Key(viewsKey(row.SpreadsheetID)).ScoreMember().ScoreMember(float64(row.Views), row.URL).Build()
	if err := q.client.Do(ctx, zCmd).Error(); err != nil {
		return fmt.Errorf("zadd: %w", err)
	}
	return nil
}

// CreateOrUpdateBatch – массовая вставка/обновление одним пайплайном (DoMulti).
func (q *Queue) CreateOrUpdateBatch(ctx context.Context, rows []*models.ResultRowUrl) error {
	if len(rows) == 0 {
		return nil
	}

	// Каждая команда собирается отдельным вызовом q.client.B() и складывается
	// в срез — valkey-go отправляет такой срез одним пайплайном через DoMulti.
	cmds := make([]valkey.Completed, 0, len(rows)*2)
	for _, row := range rows {
		data, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("marshal row %s: %w", row.URL, err)
		}
		cmds = append(cmds,
			q.client.B().Hset().Key(dataKey(row.SpreadsheetID)).FieldValue().FieldValue(row.URL, string(data)).Build(),
			q.client.B().Zadd().Key(viewsKey(row.SpreadsheetID)).ScoreMember().ScoreMember(float64(row.Views), row.URL).Build(),
		)
	}

	for _, resp := range q.client.DoMulti(ctx, cmds...) {
		if err := resp.Error(); err != nil {
			return fmt.Errorf("pipeline exec: %w", err)
		}
	}
	return nil
}

// GetAll – возвращает все записи для конкретного пользователя
func (q *Queue) GetAll(ctx context.Context, spreadsheetID string) ([]*models.ResultRowUrl, error) {
	// HGetAll возвращает map[поле]значение
	cmd := q.client.B().Hgetall().Key(dataKey(spreadsheetID)).Build()
	resp := q.client.Do(ctx, cmd)
	if resp.Error() != nil {
		// Если ключа нет, это может быть ошибка, но обычно HGetAll пустого ключа возвращает nil
		if valkey.IsValkeyNil(resp.Error()) {
			return []*models.ResultRowUrl{}, nil
		}
		return nil, fmt.Errorf("hgetall: %w", resp.Error())
	}

	// Преобразуем ответ в map
	hashMap, err := resp.AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("asstrmap: %w", err)
	}

	result := make([]*models.ResultRowUrl, 0, len(hashMap))
	for url, jsonStr := range hashMap {
		var row models.ResultRowUrl
		if err := json.Unmarshal([]byte(jsonStr), &row); err != nil {
			// Пропускаем битые записи, но можно вернуть ошибку
			continue
		}
		// Убедимся, что URL в структуре соответствует ключу (на случай рассинхрона)
		row.URL = url
		result = append(result, &row)
	}
	return result, nil
}

// GetMostViewed – возвращает записи с Views > minViews, отсортированные по убыванию
func (q *Queue) GetMostViewed(ctx context.Context, spreadsheetID string, minViews int64) ([]*models.ResultRowUrl, error) {
	// ZRevRangeByScore: возвращает элементы с score от min+1 до +inf в порядке убывания
	// Порядок аргументов у ZREVRANGEBYSCORE: key max min, поэтому и в билдере
	// сначала идёт Max(), затем Min() — иначе метод Min просто недоступен на
	// этом этапе цепочки.
	cmd := q.client.B().Zrevrangebyscore().Key(viewsKey(spreadsheetID)).
		Max("+inf").Min(strconv.FormatInt(minViews+1, 10)).
		Build()
	resp := q.client.Do(ctx, cmd)
	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return []*models.ResultRowUrl{}, nil
		}
		return nil, fmt.Errorf("zrevrangebyscore: %w", err)
	}

	// Получаем список URL (строки)
	urls, err := resp.AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("asstrslice: %w", err)
	}
	if len(urls) == 0 {
		return []*models.ResultRowUrl{}, nil
	}

	// Для каждого URL делаем HGet из hash одним пайплайном через DoMulti.
	cmds := make([]valkey.Completed, 0, len(urls))
	for _, url := range urls {
		cmds = append(cmds, q.client.B().Hget().Key(dataKey(spreadsheetID)).Field(url).Build())
	}
	responses := q.client.DoMulti(ctx, cmds...)

	result := make([]*models.ResultRowUrl, 0, len(urls))
	for i, res := range responses {
		if res.Error() != nil {
			// Если запись не найдена, пропускаем
			continue
		}
		jsonStr, err := res.ToString()
		if err != nil {
			continue
		}
		var row models.ResultRowUrl
		if err := json.Unmarshal([]byte(jsonStr), &row); err != nil {
			continue
		}
		row.URL = urls[i] // сохраняем URL из sorted set
		result = append(result, &row)
	}
	return result, nil
}

// DeleteByURL – удаление конкретной ссылки пользователя
func (q *Queue) DeleteByURL(ctx context.Context, spreadsheetID, url string) error {
	cmds := []valkey.Completed{
		q.client.B().Hdel().Key(dataKey(spreadsheetID)).Field(url).Build(),
		q.client.B().Zrem().Key(viewsKey(spreadsheetID)).Member(url).Build(),
	}

	for _, resp := range q.client.DoMulti(ctx, cmds...) {
		if err := resp.Error(); err != nil {
			return fmt.Errorf("delete by url: %w", err)
		}
	}
	return nil
}
