package models

type ColumnPositions struct {
	URLColumnIndex      int // индекс колонки "Ссылка на видео"
	CheckboxColumnIndex int // индекс колонки "Парсинг"
	CountColumnIndex    int // индекс колонки "Глубина"
}

// ResultRowUrl структура для вставки в excel таблицу
type ResultRowUrl struct {
	URL         string // Cсылка на видео
	Description string // Описание
	Views       int64  // Охват факт
	Likes       int64  // Лайки
	Comments    int64  // Комментарии
	Shares      int64  // Репосты
	ER          string // ER (likes+shares+comments)/views*100
	Virality    string // Виральность (shared/views)*100
	ParsingDate string // Дата обновления
	PublishDate string // Дата публикации
	VideoUrls   []string
}

type ResultRowAccount struct {
	URL         string // Cсылка на видео
	Description string // Описание
	Views       int64  // Охват факт
	Likes       int64  // Лайки
	Comments    int64  // Комментарии
	Shares      int64  // Репосты
	ER          string // ER (likes+shares+comments)/views*100
	Virality    string // Виральность (shared/views)*100
	ParsingDate string // Дата обновления
	PublishDate string // Дата публикации
}
