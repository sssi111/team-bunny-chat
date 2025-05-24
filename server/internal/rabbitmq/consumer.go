package rabbitmq

import (
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"time"

	"team-bunny-chat/server/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	db      *sql.DB
}

func NewConsumer(amqpURL string, db *sql.DB) (*Consumer, error) {
	log.Printf("Подключение к RabbitMQ: %s", amqpURL)
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}
	log.Printf("Успешно подключились к RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	log.Printf("Успешно создали канал")

	return &Consumer{
		conn:    conn,
		channel: ch,
		db:      db,
	}, nil
}

func (c *Consumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Consumer) SubscribeToChats() error {
	log.Printf("Создаем exchange bunny.chats")
	err := c.channel.ExchangeDeclare(
		"bunny.chats", // название exchange
		"topic",       // тип
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return err
	}
	log.Printf("Exchange успешно создан")

	q, err := c.channel.QueueDeclare(
		"",    // случайное имя
		false, // не durable
		true,  // auto-delete
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}
	log.Printf("Создана очередь: %s", q.Name)

	err = c.channel.QueueBind(
		q.Name,        // имя очереди
		"bunny.*",     // routing key
		"bunny.chats", // exchange
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return err
	}
	log.Printf("Очередь привязана к exchange с паттерном bunny.*")

	msgs, err := c.channel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return err
	}
	log.Printf("Начинаем получать сообщения")

	go c.handleMessages(msgs)

	return nil
}

func (c *Consumer) handleMessages(msgs <-chan amqp.Delivery) {
	for d := range msgs {
		parts := strings.SplitN(d.RoutingKey, ".", 2)
		if len(parts) != 2 {
			log.Printf("Некорректный routing key: %s", d.RoutingKey)
			continue
		}
		chatName := parts[1]

		var msg models.Message
		err := json.Unmarshal(d.Body, &msg)
		if err != nil {
			log.Printf("Ошибка при разборе сообщения: %v", err)
			continue
		}

		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}

		log.Printf("Получено сообщение для чата %s от пользователя %s: %s", chatName, msg.Username, msg.Body)

		err = models.SaveMessage(c.db, chatName, &msg)
		if err != nil {
			log.Printf("Ошибка при сохранении сообщения: %v", err)
		} else {
			log.Printf("Сообщение успешно сохранено в базу")
		}
	}
}
