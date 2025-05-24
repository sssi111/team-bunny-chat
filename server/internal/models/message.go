package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Message представляет сообщение в чате
type Message struct {
	Username  string    `json:"username"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

// getTableName возвращает имя таблицы для конкретного чата
func getTableName(chatName string) string {
	// Заменяем точки на подчеркивания для безопасного имени таблицы
	return fmt.Sprintf("chat_%s", strings.ReplaceAll(chatName, ".", "_"))
}

// CreateChatTable создает таблицу для конкретного чата
func CreateChatTable(db *sql.DB, chatName string) error {
	tableName := getTableName(chatName)
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		username TEXT NOT NULL,
		body TEXT NOT NULL,
		timestamp DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %s(timestamp);
	`, tableName, tableName, tableName)

	_, err := db.Exec(query)
	return err
}

// SaveMessage сохраняет сообщение в соответствующую таблицу чата
func SaveMessage(db *sql.DB, chatName string, msg *Message) error {
	// Сначала создаем таблицу, если её нет
	err := CreateChatTable(db, chatName)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	tableName := getTableName(chatName)
	query := fmt.Sprintf(`
	INSERT INTO %s (username, body, timestamp)
	VALUES (?, ?, ?)`, tableName)

	_, err = db.Exec(query, msg.Username, msg.Body, msg.Timestamp)
	if err != nil {
		return fmt.Errorf("ошибка сохранения сообщения: %w", err)
	}

	return nil
}

// GetChatMessages возвращает сообщения конкретного чата с ограничением по количеству
func GetChatMessages(db *sql.DB, chatName string, limit int) ([]Message, error) {
	tableName := getTableName(chatName)

	// Проверяем существование таблицы
	var tableCnt int
	err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&tableCnt)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки таблицы: %w", err)
	}

	// Если таблица не существует, возвращаем пустой массив
	if tableCnt == 0 {
		return []Message{}, nil
	}

	query := fmt.Sprintf(`
	SELECT username, body, timestamp
	FROM %s
	ORDER BY timestamp DESC
	LIMIT ?`, tableName)

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения сообщений: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.Username, &msg.Body, &msg.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения сообщения: %w", err)
		}
		messages = append(messages, msg)
	}

	// Разворачиваем слайс, чтобы сообщения шли в хронологическом порядке
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
