package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"team-bunny-chat/server/internal/api"
	"team-bunny-chat/server/internal/rabbitmq"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC | log.Lshortfile)
	log.Printf("=== Запуск сервера ===")

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		dbPath = "/data/chat.db"
	}

	log.Printf("Используем базу данных: %s", dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Ошибка проверки подключения к базе данных: %v", err)
	}
	log.Printf("Успешно подключились к базе данных: %s", dbPath)

	var tableCnt int
	err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='chat_aboba'").Scan(&tableCnt)
	if err != nil {
		log.Fatalf("Ошибка проверки таблицы: %v", err)
	}
	log.Printf("Найдено таблиц chat_aboba: %d", tableCnt)

	if tableCnt > 0 {
		var msgCnt int
		err = db.QueryRow("SELECT count(*) FROM chat_aboba").Scan(&msgCnt)
		if err != nil {
			log.Fatalf("Ошибка подсчета сообщений: %v", err)
		}
		log.Printf("Найдено сообщений в таблице chat_aboba: %d", msgCnt)
	}

	consumer, err := rabbitmq.NewConsumer(rabbitURL, db)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}

	err = consumer.SubscribeToChats()
	if err != nil {
		log.Fatalf("Ошибка подписки на топики: %v", err)
	}
	log.Printf("Успешно подписались на топики RabbitMQ")

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		log.Printf("=== Новый HTTP запрос ===")
		log.Printf("Метод: %s", c.Request.Method)
		log.Printf("URL: %s", c.Request.URL)
		log.Printf("Query параметры: %v", c.Request.URL.Query())

		c.Next()

		log.Printf("Статус ответа: %d", c.Writer.Status())
		log.Printf("=== Конец HTTP запроса ===")
	})

	handler := api.NewHandler(db)
	api.SetupRoutes(router, handler)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Сервер запущен на порту %s", port)
		if err := router.Run(":" + port); err != nil {
			log.Printf("Ошибка запуска сервера: %v", err)
			sigChan <- syscall.SIGTERM
		}
	}()

	<-sigChan
	log.Println("Получен сигнал завершения")

	consumer.Close()
	db.Close()
	log.Println("Соединения закрыты")
}
