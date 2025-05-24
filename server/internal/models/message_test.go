package models

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func SetupTestDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		return nil, err
	}
	return db, nil
}

func TestCreateChatTable(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err)
	defer func() {
		db.Close()
		os.Remove("test.db")
	}()

	tests := []struct {
		name     string
		chatName string
		wantErr  bool
	}{
		{
			name:     "Создание новой таблицы",
			chatName: "test_chat",
			wantErr:  false,
		},
		{
			name:     "Повторное создание таблицы",
			chatName: "test_chat",
			wantErr:  false,
		},
		{
			name:     "Создание таблицы с специальными символами",
			chatName: "test.chat.special",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateChatTable(db, tt.chatName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				var tableName string
				err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?",
					getTableName(tt.chatName)).Scan(&tableName)
				assert.NoError(t, err)
				assert.Equal(t, getTableName(tt.chatName), tableName)
			}
		})
	}
}

func TestSaveAndGetMessages(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err)
	defer func() {
		db.Close()
		os.Remove("test.db")
	}()

	chatName := "test_chat"
	messages := []Message{
		{
			Username:  "user1",
			Body:      "Message 1",
			Timestamp: time.Now().UTC(),
		},
		{
			Username:  "user2",
			Body:      "Message 2",
			Timestamp: time.Now().UTC(),
		},
	}

	for _, msg := range messages {
		err := SaveMessage(db, chatName, &msg)
		assert.NoError(t, err)
	}

	retrieved, err := GetChatMessages(db, chatName, 10)
	assert.NoError(t, err)
	assert.Len(t, retrieved, len(messages))

	for i, msg := range retrieved {
		assert.Equal(t, messages[i].Username, msg.Username)
		assert.Equal(t, messages[i].Body, msg.Body)
		assert.Equal(t, messages[i].Timestamp.Format("2006-01-02 15:04:05"),
			msg.Timestamp.Format("2006-01-02 15:04:05"))
	}
}

func TestGetChatMessagesLimit(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err)
	defer func() {
		db.Close()
		os.Remove("test.db")
	}()

	chatName := "test_chat"

	for i := 0; i < 5; i++ {
		msg := &Message{
			Username:  "user",
			Body:      "Message",
			Timestamp: time.Now().UTC(),
		}
		err := SaveMessage(db, chatName, msg)
		assert.NoError(t, err)
	}

	tests := []struct {
		name      string
		limit     int
		wantCount int
	}{
		{
			name:      "Получение всех сообщений",
			limit:     10,
			wantCount: 5,
		},
		{
			name:      "Получение с лимитом меньше количества сообщений",
			limit:     3,
			wantCount: 3,
		},
		{
			name:      "Получение с лимитом 1",
			limit:     1,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := GetChatMessages(db, chatName, tt.limit)
			assert.NoError(t, err)
			assert.Len(t, messages, tt.wantCount)
		})
	}
}

func TestGetChatMessagesNonExistentChat(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err)
	defer func() {
		db.Close()
		os.Remove("test.db")
	}()

	messages, err := GetChatMessages(db, "nonexistent_chat", 10)
	assert.NoError(t, err)
	assert.Empty(t, messages)
}
