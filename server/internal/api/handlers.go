package api

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"team-bunny-chat/server/internal/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) GetChatHistory(c *gin.Context) {
	log.Printf("=== Начало обработки запроса GetChatHistory ===")
	chat := c.Query("chat")
	if chat == "" {
		log.Printf("Отсутствует параметр chat")
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat parameter is required"})
		return
	}

	log.Printf("Получен запрос на историю чата: %s", chat)

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	log.Printf("Запрашиваем %d сообщений", limit)

	err = h.db.Ping()
	if err != nil {
		log.Printf("Ошибка проверки подключения к базе данных: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection error"})
		return
	}
	log.Printf("Подключение к базе данных активно")

	var tableCnt int
	err = h.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", "chat_"+chat).Scan(&tableCnt)
	if err != nil {
		log.Printf("Ошибка проверки таблицы: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	log.Printf("Найдено таблиц: %d", tableCnt)

	if tableCnt > 0 {
		var msgCnt int
		err = h.db.QueryRow("SELECT count(*) FROM chat_" + chat).Scan(&msgCnt)
		if err != nil {
			log.Printf("Ошибка подсчета сообщений: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		log.Printf("Найдено сообщений: %d", msgCnt)
	}

	messages, err := models.GetChatMessages(h.db, chat, limit)
	if err != nil {
		log.Printf("Ошибка при получении сообщений: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Printf("Получено %d сообщений для чата %s", len(messages), chat)
	for i, msg := range messages {
		log.Printf("Сообщение %d: %+v", i+1, msg)
	}
	log.Printf("=== Конец обработки запроса GetChatHistory ===")

	c.JSON(http.StatusOK, gin.H{
		"chat":     chat,
		"messages": messages,
	})
}

func SetupRoutes(router *gin.Engine, handler *Handler) {
	v1 := router.Group("/v1")
	{
		v1.GET("/chats/history", handler.GetChatHistory)
	}
}
