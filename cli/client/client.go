package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	Username  string    `json:"username"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

type Client struct {
	conn         *amqp.Connection
	ch           *amqp.Channel
	username     string
	chatName     string
	messages     chan Message
	closeConn    chan struct{}
	queue        amqp.Queue
	rabbitmqHost string
}

func NewClient(username, chatName, rabbitmqHost string) (*Client, error) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://guest:guest@%s/", rabbitmqHost))
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ошибка создания канала: %v", err)
	}

	client := &Client{
		conn:         conn,
		ch:           ch,
		username:     username,
		chatName:     chatName,
		messages:     make(chan Message),
		closeConn:    make(chan struct{}),
		rabbitmqHost: rabbitmqHost,
	}

	if err := client.setup(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

func (c *Client) setup() error {
	err := c.ch.ExchangeDeclare(
		"bunny.chats",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("ошибка объявления exchange: %v", err)
	}

	q, err := c.ch.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("ошибка создания очереди: %v", err)
	}

	c.queue = q

	err = c.ch.QueueBind(
		q.Name,
		c.chatName,
		"bunny.chats",
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("ошибка привязки очереди: %v", err)
	}

	msgs, err := c.ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("ошибка подписки на сообщения: %v", err)
	}

	go c.handleMessages(msgs)

	return nil
}

func (c *Client) handleMessages(msgs <-chan amqp.Delivery) {
	for {
		select {
		case msg := <-msgs:
			var message Message
			if err := json.Unmarshal(msg.Body, &message); err != nil {
				log.Printf("Ошибка разбора сообщения: %v", err)
				continue
			}
			c.messages <- message
		case <-c.closeConn:
			return
		}
	}
}

func (c *Client) SendMessage(text string) error {
	msg := Message{
		Username:  c.username,
		Body:      text,
		Timestamp: time.Now(),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("ошибка сериализации сообщения: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.ch.PublishWithContext(ctx,
		"bunny.chats",
		c.chatName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения: %v", err)
	}

	return nil
}

func (c *Client) SwitchChat(newChatName string) error {
	err := c.ch.QueueUnbind(
		c.queue.Name,
		c.chatName,
		"bunny.chats",
		nil,
	)
	if err != nil {
		return fmt.Errorf("ошибка отвязки от старого чата: %v", err)
	}

	err = c.ch.QueueBind(
		c.queue.Name,
		newChatName,
		"bunny.chats",
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("ошибка привязки к новому чату: %v", err)
	}

	c.chatName = newChatName

	return nil
}

func (c *Client) Messages() <-chan Message {
	return c.messages
}

func (c *Client) GetCurrentChat() string {
	return c.chatName
}

func (c *Client) Close() {
	close(c.closeConn)
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
} 