package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
)

func TestMessage(t *testing.T) {
	msg := Message{
		Username:  "testuser",
		Body:      "Test message",
		Timestamp: time.Now(),
		ChatName:  "testchat",
	}
	
	assert.Equal(t, "testuser", msg.Username)
	assert.Equal(t, "Test message", msg.Body)
	assert.Equal(t, "testchat", msg.ChatName)
}

func TestHistoryResponse(t *testing.T) {
	msg := Message{
		Username:  "testuser",
		Body:      "Test message",
		Timestamp: time.Now(),
		ChatName:  "testchat",
	}
	
	history := HistoryResponse{
		Chat:     "testchat",
		Messages: []Message{msg},
	}
	
	assert.Equal(t, "testchat", history.Chat)
	assert.Equal(t, 1, len(history.Messages))
	assert.Equal(t, msg.Username, history.Messages[0].Username)
	assert.Equal(t, msg.Body, history.Messages[0].Body)
}

func TestNewChatApp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	
	chatApp := NewChatApp()
	
	assert.NotNil(t, chatApp)
	assert.NotNil(t, chatApp.app)
	assert.NotNil(t, chatApp.window)
	assert.Equal(t, defaultChat, chatApp.currentChannel)
	assert.Equal(t, fmt.Sprintf("http://%s", defaultServer), chatApp.serverURL)
	assert.Equal(t, fmt.Sprintf("amqp://guest:guest@%s/", defaultRabbitMQ), chatApp.rabbitMQURL)
}

func TestInitUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	
	app := test.NewApp()
	w := app.NewWindow(appTitle)
	
	chatApp := &ChatApp{
		app:            app,
		window:         w,
		messages:       make([]Message, 0),
		username:       "testuser",
		currentChannel: defaultChat,
		serverURL:      fmt.Sprintf("http://%s", defaultServer),
		rabbitMQURL:    fmt.Sprintf("amqp://guest:guest@%s/", defaultRabbitMQ),
	}
	
	chatApp.initUI()
	
	assert.NotNil(t, chatApp.messageList)
	assert.NotNil(t, chatApp.messageInput)
	assert.NotNil(t, chatApp.sendButton)
	assert.NotNil(t, chatApp.channelSelect)
	assert.NotNil(t, chatApp.usernameLabel)
	assert.NotNil(t, chatApp.statusLabel)
	
	assert.Equal(t, "Пользователь: testuser", chatApp.usernameLabel.Text)
	
	assert.Equal(t, []string{defaultChat}, chatApp.channelSelect.Options)
	assert.Equal(t, defaultChat, chatApp.channelSelect.Selected)
}

func TestAddMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	
	app := test.NewApp()
	w := app.NewWindow(appTitle)
	
	chatApp := &ChatApp{
		app:            app,
		window:         w,
		messages:       make([]Message, 0),
		username:       "testuser",
		currentChannel: defaultChat,
		serverURL:      fmt.Sprintf("http://%s", defaultServer),
		rabbitMQURL:    fmt.Sprintf("amqp://guest:guest@%s/", defaultRabbitMQ),
	}
	
	chatApp.initUI()
	
	msg := Message{
		Username:  "testuser",
		Body:      "Test message",
		Timestamp: time.Now(),
		ChatName:  defaultChat,
	}
	
	chatApp.addMessage(msg)
	
	time.Sleep(100 * time.Millisecond)
	
	assert.Equal(t, 1, len(chatApp.messages))
	assert.Equal(t, msg.Username, chatApp.messages[0].Username)
	assert.Equal(t, msg.Body, chatApp.messages[0].Body)
	assert.Equal(t, msg.ChatName, chatApp.messages[0].ChatName)
}

func TestSetStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	
	app := test.NewApp()
	w := app.NewWindow(appTitle)
	
	chatApp := &ChatApp{
		app:            app,
		window:         w,
		messages:       make([]Message, 0),
		username:       "testuser",
		currentChannel: defaultChat,
		serverURL:      fmt.Sprintf("http://%s", defaultServer),
		rabbitMQURL:    fmt.Sprintf("amqp://guest:guest@%s/", defaultRabbitMQ),
	}
	
	chatApp.initUI()
	
	chatApp.setStatus("Test status")
	
	time.Sleep(100 * time.Millisecond)
	
	assert.Equal(t, "Статус: Test status", chatApp.statusLabel.Text)
}

func TestFetchHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	
	app := test.NewApp()
	w := app.NewWindow(appTitle)
	
	chatApp := &ChatApp{
		app:            app,
		window:         w,
		messages:       make([]Message, 0),
		username:       "testuser",
		currentChannel: defaultChat,
		serverURL:      fmt.Sprintf("http://%s", defaultServer),
		rabbitMQURL:    fmt.Sprintf("amqp://guest:guest@%s/", defaultRabbitMQ),
	}
	
	chatApp.initUI()
	
	historyMessages := []Message{
		{
			Username:  "user1",
			Body:      "Message 1",
			Timestamp: time.Now().Add(-2 * time.Minute),
			ChatName:  defaultChat,
		},
		{
			Username:  "user2",
			Body:      "Message 2",
			Timestamp: time.Now().Add(-1 * time.Minute),
			ChatName:  defaultChat,
		},
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		history := HistoryResponse{
			Chat:     defaultChat,
			Messages: historyMessages,
		}
		json.NewEncoder(w).Encode(history)
	}))
	defer server.Close()
	
	chatApp.serverURL = server.URL
	
	chatApp.fetchHistory()
	
	time.Sleep(100 * time.Millisecond)
	
	assert.Equal(t, 2, len(chatApp.messages))
	assert.Equal(t, historyMessages[0].Username, chatApp.messages[0].Username)
	assert.Equal(t, historyMessages[0].Body, chatApp.messages[0].Body)
	assert.Equal(t, historyMessages[1].Username, chatApp.messages[1].Username)
	assert.Equal(t, historyMessages[1].Body, chatApp.messages[1].Body)
}