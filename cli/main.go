package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"team-bunny-chat/client"
)

func main() {
	username := flag.String("user", "", "Имя пользователя")
	chatName := flag.String("chat", "", "Название чата")
	rabbitmqHost := flag.String("rabbitmq", "localhost:5672", "Адрес RabbitMQ сервера")
	serverHost := flag.String("server", "localhost:8080", "Адрес сервера чата")
	flag.Parse()

	if *username == "" || *chatName == "" {
		log.Fatal("Необходимо указать имя пользователя (-user) и название чата (-chat)")
	}

	fmt.Println("Получение истории сообщений...")
	history, err := rabbitmq.GetChatHistory(*chatName, *serverHost)
	if err != nil {
		log.Printf("Ошибка при получении истории: %v", err)
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

	client, err := rabbitmq.NewClient(*username, *chatName, *rabbitmqHost)
	if err != nil {
		log.Fatalf("Ошибка создания клиента: %v", err)
	}
	defer client.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range client.Messages() {
			fmt.Printf("\n[%s] %s: %s\n> ",
				msg.Timestamp.Format("15:04:05"),
				msg.Username,
				msg.Body)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("> Вы находитесь в чате: %s\n> ", client.GetCurrentChat())

	go func() {
		for scanner.Scan() {
			text := scanner.Text()
			
			if strings.HasPrefix(text, "/") {
				parts := strings.Fields(text)
				command := parts[0]
				
				switch command {
				case "/quit":
					sigChan <- syscall.SIGINT
					return
				case "/switch":
					if len(parts) < 2 {
						fmt.Println("Использование: /switch <название_чата>")
						fmt.Print("> ")
						continue
					}
					newChat := parts[1]
					
					if err := client.SwitchChat(newChat); err != nil {
						log.Printf("Ошибка при переключении чата: %v", err)
					} else {
						fmt.Printf("Вы переключились в чат: %s\n", newChat)
					}
					
					fmt.Printf("\nПолучение истории чата %s...\n", newChat)
					history, err := rabbitmq.GetChatHistory(newChat, *serverHost)
					if err != nil {
						log.Printf("Ошибка при получении истории: %v", err)
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
				default:
					fmt.Printf("Неизвестная команда: %s\nДоступные команды:\n/switch <чат> - сменить чат\n/quit - выйти\n", command)
				}
			} else {
				if err := client.SendMessage(text); err != nil {
					log.Printf("Ошибка отправки сообщения: %v", err)
				}
			}

			fmt.Print("> ")
		}
	}()

	<-sigChan
	fmt.Println("\nЗавершение работы...")
}
