package tags

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Tag — один тег в ответе API.
type Tag struct {
	Tag   string `json:"tag"`
	Users uint32 `json:"users"`
}

// Store — всё, что обработчику нужно знать о хранилище тегов.
// Интерфейс позволит в тестах подставить заглушку вместо настоящей базы.
type Store interface {
	Tags(ctx context.Context, query string) ([]Tag, error)
}

// ClickHouseStore читает теги из таблицы query_tags.
type ClickHouseStore struct {
	conn    driver.Conn
	timeout time.Duration
}

func NewClickHouseStore(conn driver.Conn, timeout time.Duration) *ClickHouseStore {
	return &ClickHouseStore{conn: conn, timeout: timeout}
}

// Значение запроса передаётся отдельно от текста SQL: вместо ? драйвер подставит
// параметр сам, с экранированием. Склейки строк в нашем коде нет.
const tagsSQL = `SELECT tag, users FROM query_tags WHERE query = ? ORDER BY rank`

func (s *ClickHouseStore) Tags(ctx context.Context, query string) ([]Tag, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	rows, err := s.conn.Query(ctx, tagsSQL, query)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	defer rows.Close()

	// make(...) создаёт пустой, но не nil-срез: в JSON он превратится в [], а не в null.
	result := make([]Tag, 0, 10)
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.Tag, &t.Users); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return result, nil
}
