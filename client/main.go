package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	Username  string    `json:"username"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

type HistoryResponse struct {
	Chat     string    `json:"chat"`
	Messages []Message `json:"messages"`
}

func main() {
	// Парсим аргументы командной строки
	username := flag.String("user", "", "Имя пользователя")
	chatName := flag.String("chat", "", "Название чата")
	flag.Parse()

	if *username == "" || *chatName == "" {
		log.Fatal("Необходимо указать имя пользователя (-user) и название чата (-chat)")
	}

	// Получаем историю сообщений
	fmt.Println("Получение истории сообщений...")
	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/v1/chats/history?chat=%s", *chatName))
	if err != nil {
		log.Printf("Ошибка при получении истории: %v", err)
	} else {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Ошибка при чтении ответа: %v", err)
		} else {
			var history HistoryResponse
			if err := json.Unmarshal(body, &history); err != nil {
				log.Printf("Ошибка при разборе истории: %v", err)
			} else {
				if len(history.Messages) > 0 {
					fmt.Println("\nИстория сообщений:")
					for _, msg := range history.Messages {
						fmt.Printf("[%s] %s: %s\n",
							msg.Timestamp.Format("15:04:05"),
							msg.Username,
							msg.Body)
					}
					fmt.Println("\nКонец истории")
				} else {
					fmt.Println("История сообщений пуста")
				}
			}
		}
	}

	// Подключаемся к RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Создаем канал
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Ошибка создания канала: %v", err)
	}
	defer ch.Close()

	// Объявляем exchange
	err = ch.ExchangeDeclare(
		"bunny.chats", // название
		"topic",       // тип
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Fatalf("Ошибка объявления exchange: %v", err)
	}

	// Создаем очередь для получения сообщений
	q, err := ch.QueueDeclare(
		"",    // название (пустое для автогенерации)
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Ошибка создания очереди: %v", err)
	}

	// Привязываем очередь к exchange
	err = ch.QueueBind(
		q.Name,        // название очереди
		*chatName,     // routing key
		"bunny.chats", // exchange
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Fatalf("Ошибка привязки очереди: %v", err)
	}

	// Создаем канал для получения сообщений
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Ошибка подписки на сообщения: %v", err)
	}

	// Канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Горутина для получения сообщений
	go func() {
		for msg := range msgs {
			var message Message
			if err := json.Unmarshal(msg.Body, &message); err != nil {
				log.Printf("Ошибка разбора сообщения: %v", err)
				continue
			}
			fmt.Printf("\n[%s] %s: %s\n> ",
				message.Timestamp.Format("15:04:05"),
				message.Username,
				message.Body)
		}
	}()

	// Читаем ввод пользователя
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	go func() {
		for scanner.Scan() {
			text := scanner.Text()
			if text == "/quit" {
				sigChan <- syscall.SIGINT
				return
			}

			// Создаем сообщение
			msg := Message{
				Username:  *username,
				Body:      text,
				Timestamp: time.Now(),
			}

			// Сериализуем сообщение
			body, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Ошибка сериализации сообщения: %v", err)
				continue
			}

			// Отправляем сообщение
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = ch.PublishWithContext(ctx,
				"bunny.chats", // exchange
				*chatName,     // routing key
				false,         // mandatory
				false,         // immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        body,
				})
			cancel()

			if err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}

			fmt.Print("> ")
		}
	}()

	// Ожидаем сигнал завершения
	<-sigChan
	fmt.Println("\nЗавершение работы...")
}
