package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	appTitle        = "Team Bunny Chat"
	defaultServer   = "localhost:8080"
	defaultRabbitMQ = "localhost:5672"
	defaultChat     = "general"
	exchangeName    = "bunny.chats"
)

type Message struct {
	Username  string    `json:"username"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
	ChatName  string    `json:"chat_name,omitempty"`
}

type HistoryResponse struct {
	Chat     string    `json:"chat"`
	Messages []Message `json:"messages"`
}

type ChatApp struct {
	app    fyne.App
	window fyne.Window

	messageList   *widget.List
	messageInput  *widget.Entry
	sendButton    *widget.Button
	channelSelect *widget.Select
	usernameLabel *widget.Label
	statusLabel   *widget.Label

	messages       []Message
	username       string
	currentChannel string
	serverURL      string
	rabbitMQURL    string

	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func getServerURL() string {
	if server := os.Getenv("CHAT_SERVER"); server != "" {
		return fmt.Sprintf("http://%s", server)
	}
	return fmt.Sprintf("http://%s", defaultServer)
}

func getRabbitMQURL() string {
	if rabbitMQ := os.Getenv("CHAT_RABBITMQ"); rabbitMQ != "" {
		return fmt.Sprintf("amqp://guest:guest@%s/", rabbitMQ)
	}
	return fmt.Sprintf("amqp://guest:guest@%s/", defaultRabbitMQ)
}

func NewChatApp() *ChatApp {
	a := app.New()
	a.Settings().SetTheme(theme.DarkTheme())

	w := a.NewWindow(appTitle)
	w.Resize(fyne.NewSize(800, 600))

	chatApp := &ChatApp{
		app:            a,
		window:         w,
		messages:       make([]Message, 0),
		username:       os.Getenv("USER"),
		currentChannel: defaultChat,
		serverURL:      getServerURL(),
		rabbitMQURL:    getRabbitMQURL(),
	}

	chatApp.initUI()

	return chatApp
}

func (c *ChatApp) initUI() {
	c.messageList = widget.NewList(
		func() int {
			return len(c.messages)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(""),
				canvas.NewLine(theme.ForegroundColor()),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(c.messages) {
				return
			}
			
			message := c.messages[id]
			vbox := obj.(*fyne.Container)
			
			header := vbox.Objects[0].(*widget.Label)
			header.SetText(fmt.Sprintf("[%s] %s", 
				message.Timestamp.Format("15:04:05"),
				message.Username))
			
			body := vbox.Objects[1].(*widget.Label)
			body.SetText(message.Body)
			
			line := vbox.Objects[2].(*canvas.Line)
			line.StrokeColor = theme.DisabledColor()
			line.StrokeWidth = 1
		},
	)

	c.messageInput = widget.NewMultiLineEntry()
	c.messageInput.SetPlaceHolder("Введите сообщение...")
	c.messageInput.OnSubmitted = func(text string) {
		c.sendMessage(text)
	}

	c.sendButton = widget.NewButtonWithIcon("Отправить", theme.MailSendIcon(), func() {
		c.sendMessage(c.messageInput.Text)
	})

	c.channelSelect = widget.NewSelect([]string{defaultChat}, func(selected string) {
		if selected != c.currentChannel {
			c.switchChannel(selected)
		}
	})
	c.channelSelect.SetSelected(defaultChat)

	c.usernameLabel = widget.NewLabelWithStyle(
		fmt.Sprintf("Пользователь: %s", c.username),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	c.statusLabel = widget.NewLabel("Статус: Отключен")

	addChannelButton := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		entry := widget.NewEntry()
		entry.SetPlaceHolder("Введите название канала")
		
		dialog.ShowCustomConfirm("Новый канал", "Добавить", "Отмена", entry, func(ok bool) {
			if ok && entry.Text != "" {
				channelName := strings.TrimSpace(entry.Text)
				options := append(c.channelSelect.Options, channelName)
				c.channelSelect.SetOptions(options)
				c.channelSelect.SetSelected(channelName)
			}
		}, c.window)
	})

	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		c.showSettingsDialog()
	})

	topBar := container.NewBorder(
		nil, nil, 
		container.NewHBox(
			widget.NewLabel("Канал:"),
			c.channelSelect,
			addChannelButton,
		),
		container.NewHBox(
			c.usernameLabel,
			settingsButton,
		),
	)

	bottomBar := container.NewBorder(
		nil, nil, nil, 
		container.NewHBox(
			c.sendButton,
		),
		c.messageInput,
	)

	content := container.NewBorder(
		topBar,
		container.NewBorder(
			nil, bottomBar, nil, nil,
			container.NewVBox(
				c.statusLabel,
			),
		),
		nil, nil,
		container.NewPadded(c.messageList),
	)

	c.window.SetContent(content)
}

func (c *ChatApp) showSettingsDialog() {
	usernameEntry := widget.NewEntry()
	usernameEntry.SetText(c.username)
	
	serverEntry := widget.NewEntry()
	serverEntry.SetText(strings.TrimPrefix(c.serverURL, "http://"))
	
	rabbitMQEntry := widget.NewEntry()
	rabbitMQEntry.SetText(strings.Split(strings.TrimPrefix(c.rabbitMQURL, "amqp://guest:guest@"), "/")[0])
	
	form := widget.NewForm(
		widget.NewFormItem("Имя пользователя", usernameEntry),
		widget.NewFormItem("Сервер истории", serverEntry),
		widget.NewFormItem("Сервер RabbitMQ", rabbitMQEntry),
	)
	
	dialog.ShowCustomConfirm("Настройки", "Сохранить", "Отмена", form, func(ok bool) {
		if ok {
			newUsername := usernameEntry.Text
			newServer := serverEntry.Text
			newRabbitMQ := rabbitMQEntry.Text
			
			reconnectNeeded := false
			
			if newUsername != c.username {
				c.username = newUsername
				c.usernameLabel.SetText(fmt.Sprintf("Пользователь: %s", c.username))
			}
			
			if newServer != strings.TrimPrefix(c.serverURL, "http://") {
				c.serverURL = fmt.Sprintf("http://%s", newServer)
				reconnectNeeded = true
			}
			
			if newRabbitMQ != strings.Split(strings.TrimPrefix(c.rabbitMQURL, "amqp://guest:guest@"), "/")[0] {
				c.rabbitMQURL = fmt.Sprintf("amqp://guest:guest@%s/", newRabbitMQ)
				reconnectNeeded = true
			}
			
			if reconnectNeeded && c.conn != nil {
				c.disconnect()
				c.connect()
			}
		}
	}, c.window)
}

func (c *ChatApp) Run() {
	c.connect()
	c.fetchHistory()
	c.window.ShowAndRun()
	c.disconnect()
}

func (c *ChatApp) connect() {
	var err error
	
	c.conn, err = amqp.Dial(c.rabbitMQURL)
	if err != nil {
		c.setStatus(fmt.Sprintf("Ошибка подключения к RabbitMQ: %v", err))
		return
	}
	
	c.channel, err = c.conn.Channel()
	if err != nil {
		c.setStatus(fmt.Sprintf("Ошибка создания канала: %v", err))
		return
	}
	
	err = c.channel.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		c.setStatus(fmt.Sprintf("Ошибка объявления exchange: %v", err))
		return
	}
	
	c.queue, err = c.channel.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		c.setStatus(fmt.Sprintf("Ошибка создания очереди: %v", err))
		return
	}
	
	err = c.channel.QueueBind(
		c.queue.Name,
		c.currentChannel,
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		c.setStatus(fmt.Sprintf("Ошибка привязки очереди: %v", err))
		return
	}
	
	msgs, err := c.channel.Consume(
		c.queue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		c.setStatus(fmt.Sprintf("Ошибка подписки на сообщения: %v", err))
		return
	}
	
	go func() {
		for msg := range msgs {
			var message Message
			if err := json.Unmarshal(msg.Body, &message); err != nil {
				log.Printf("Ошибка разбора сообщения: %v", err)
				continue
			}
			
			c.window.Canvas().Refresh(c.messageList)
			c.addMessage(message)
		}
	}()
	
	c.setStatus("Подключено")
}

func (c *ChatApp) disconnect() {
	if c.channel != nil {
		c.channel.Close()
	}
	
	if c.conn != nil {
		c.conn.Close()
	}
	
	c.setStatus("Отключено")
}

func (c *ChatApp) fetchHistory() {
	go func() {
		c.setStatus("Получение истории сообщений...")
		
		url := fmt.Sprintf("%s/v1/chats/history?chat=%s", c.serverURL, c.currentChannel)
		
		resp, err := http.Get(url)
		if err != nil {
			c.setStatus(fmt.Sprintf("Ошибка при получении истории: %v", err))
			return
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.setStatus(fmt.Sprintf("Ошибка при чтении ответа: %v", err))
			return
		}
		
		var history HistoryResponse
		if err := json.Unmarshal(body, &history); err != nil {
			c.setStatus(fmt.Sprintf("Ошибка при разборе истории: %v", err))
			return
		}
		
		c.messages = make([]Message, 0, len(history.Messages))
		
		for _, msg := range history.Messages {
			c.addMessage(msg)
		}
		
		c.setStatus(fmt.Sprintf("Получено %d сообщений из истории", len(history.Messages)))
		c.window.Canvas().Refresh(c.messageList)
	}()
}

func (c *ChatApp) switchChannel(channelName string) {
	if c.channel != nil {
		err := c.channel.QueueUnbind(
			c.queue.Name,
			c.currentChannel,
			exchangeName,
			nil,
		)
		if err != nil {
			c.setStatus(fmt.Sprintf("Ошибка отвязки очереди: %v", err))
			return
		}
		
		err = c.channel.QueueBind(
			c.queue.Name,
			channelName,
			exchangeName,
			false,
			nil,
		)
		if err != nil {
			c.setStatus(fmt.Sprintf("Ошибка привязки очереди: %v", err))
			return
		}
	}
	
	c.currentChannel = channelName
	
	c.messages = make([]Message, 0)
	c.messageList.Refresh()
	
	c.fetchHistory()
	
	c.setStatus(fmt.Sprintf("Переключено на канал: %s", channelName))
}

func (c *ChatApp) sendMessage(text string) {
	if text == "" {
		return
	}
	
	c.messageInput.SetText("")
	
	if c.channel == nil {
		c.setStatus("Нет подключения к серверу")
		return
	}
	
	msg := Message{
		Username:  c.username,
		Body:      text,
		Timestamp: time.Now(),
		ChatName:  c.currentChannel,
	}
	
	body, err := json.Marshal(msg)
	if err != nil {
		c.setStatus(fmt.Sprintf("Ошибка сериализации сообщения: %v", err))
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = c.channel.PublishWithContext(ctx,
		exchangeName,
		c.currentChannel,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	
	if err != nil {
		c.setStatus(fmt.Sprintf("Ошибка отправки сообщения: %v", err))
		return
	}
}

func (c *ChatApp) addMessage(msg Message) {
	c.messages = append(c.messages, msg)
	c.window.Canvas().Refresh(c.messageList)
	c.messageList.Refresh()
	c.messageList.ScrollToBottom()
}

func (c *ChatApp) setStatus(status string) {
	statusText := fmt.Sprintf("Статус: %s", status)
	c.statusLabel.SetText(statusText)
	c.window.Canvas().Refresh(c.statusLabel)
}

func main() {
	serverFlag := flag.String("server", "", "Server address (e.g., localhost:8080)")
	rabbitMQFlag := flag.String("rabbitmq", "", "RabbitMQ address (e.g., localhost:5672)")
	flag.Parse()

	if *serverFlag != "" {
		os.Setenv("CHAT_SERVER", *serverFlag)
	}
	if *rabbitMQFlag != "" {
		os.Setenv("CHAT_RABBITMQ", *rabbitMQFlag)
	}

	chatApp := NewChatApp()
	chatApp.Run()
}