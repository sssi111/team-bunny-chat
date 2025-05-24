package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Message struct {
	Username  string    `json:"username"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

func getTableName(chatName string) string {
	return fmt.Sprintf("chat_%s", strings.ReplaceAll(chatName, ".", "_"))
}

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

func SaveMessage(db *sql.DB, chatName string, msg *Message) error {
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

func GetChatMessages(db *sql.DB, chatName string, limit int) ([]Message, error) {
	tableName := getTableName(chatName)

	var tableCnt int
	err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&tableCnt)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки таблицы: %w", err)
	}

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

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
