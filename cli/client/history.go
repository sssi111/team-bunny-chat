package rabbitmq

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HistoryResponse struct {
	Chat     string    `json:"chat"`
	Messages []Message `json:"messages"`
}

func GetChatHistory(chatName, serverHost string) (*HistoryResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/v1/chats/history?chat=%s", serverHost, chatName))
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении истории: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении ответа: %v", err)
	}

	var history HistoryResponse
	if err := json.Unmarshal(body, &history); err != nil {
		return nil, fmt.Errorf("ошибка при разборе истории: %v", err)
	}

	return &history, nil
} 