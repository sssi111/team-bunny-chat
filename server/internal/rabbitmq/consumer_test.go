package rabbitmq

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"team-bunny-chat/server/internal/models"

	_ "github.com/mattn/go-sqlite3"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		t.Fatalf("Ошибка создания тестовой БД: %v", err)
	}
	return db
}

func setupTestRabbitMQ(t *testing.T) (*amqp.Connection, *amqp.Channel, string) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Skipf("RabbitMQ не доступен: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		t.Fatalf("Ошибка создания канала: %v", err)
	}

	exchangeName := "test.exchange"
	err = ch.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		t.Fatalf("Ошибка создания exchange: %v", err)
	}

	return conn, ch, exchangeName
}

func TestConsumerMessageHandling(t *testing.T) {
	conn, ch, exchangeName := setupTestRabbitMQ(t)
	defer conn.Close()
	defer ch.Close()

	db := setupTestDB(t)
	defer func() {
		db.Close()
	}()

	consumer, err := NewConsumer("amqp://guest:guest@localhost:5672/", db)
	assert.NoError(t, err)
	defer consumer.Close()

	q, err := ch.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	assert.NoError(t, err)

	err = ch.QueueBind(
		q.Name,
		"test.chat",
		exchangeName,
		false,
		nil,
	)
	assert.NoError(t, err)

	testMessage := models.Message{
		Username:  "test_user",
		Body:      "Test message",
		Timestamp: time.Now().UTC(),
	}

	body, err := json.Marshal(testMessage)
	assert.NoError(t, err)

	err = ch.PublishWithContext(
		context.Background(),
		exchangeName,
		"test.chat",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	assert.NoError(t, err)

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	assert.NoError(t, err)

	select {
	case msg := <-msgs:
		var receivedMsg models.Message
		err := json.Unmarshal(msg.Body, &receivedMsg)
		assert.NoError(t, err)
		assert.Equal(t, testMessage.Username, receivedMsg.Username)
		assert.Equal(t, testMessage.Body, receivedMsg.Body)
	case <-time.After(5 * time.Second):
		t.Fatal("Таймаут при ожидании сообщения")
	}
}

func TestConsumerReconnection(t *testing.T) {
	conn, ch, _ := setupTestRabbitMQ(t)
	defer conn.Close()
	defer ch.Close()

	db := setupTestDB(t)
	defer func() {
		db.Close()
	}()

	consumer, err := NewConsumer("amqp://guest:guest@localhost:5672/", db)
	assert.NoError(t, err)

	consumer.conn.Close()

	newConn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	assert.NoError(t, err)
	consumer.conn = newConn

	newCh, err := newConn.Channel()
	assert.NoError(t, err)
	consumer.channel = newCh

	assert.NotNil(t, consumer.channel)
}
