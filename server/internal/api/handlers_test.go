package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"team-bunny-chat/server/internal/models"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) (*Handler, func()) {
	// Создаем временную базу данных для тестов
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		t.Fatalf("Ошибка создания тестовой БД: %v", err)
	}

	handler := NewHandler(db)

	// Функция очистки
	cleanup := func() {
		db.Close()
		os.Remove("test.db")
	}

	return handler, cleanup
}

func TestGetChatHistory(t *testing.T) {
	handler, cleanup := setupTestDB(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/chats/history", handler.GetChatHistory)

	tests := []struct {
		name           string
		chatName       string
		expectedStatus int
		setupFunc      func()
	}{
		{
			name:           "Пустая история чата",
			chatName:       "test_chat",
			expectedStatus: http.StatusOK,
			setupFunc:      func() {},
		},
		{
			name:           "Чат с сообщениями",
			chatName:       "test_chat_with_messages",
			expectedStatus: http.StatusOK,
			setupFunc: func() {
				msg := &models.Message{
					Username: "test_user",
					Body:     "Test message",
				}
				err := models.SaveMessage(handler.db, "test_chat_with_messages", msg)
				assert.NoError(t, err)
			},
		},
		{
			name:           "Отсутствующий параметр chat",
			chatName:       "",
			expectedStatus: http.StatusBadRequest,
			setupFunc:      func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			w := httptest.NewRecorder()
			url := "/v1/chats/history"
			if tt.chatName != "" {
				url += "?chat=" + tt.chatName
			}
			req, _ := http.NewRequest("GET", url, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response struct {
					Chat     string           `json:"chat"`
					Messages []models.Message `json:"messages"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.chatName, response.Chat)
			}
		})
	}
}
