package tg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const apiBase = "https://api.telegram.org/bot"

type Client struct {
	botToken string
	chatID   string
	http     *http.Client
}

func NewClient(botToken, chatID string) *Client {
	return &Client{
		botToken: botToken,
		chatID:   chatID,
		http:     &http.Client{},
	}
}

type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendMessage sends text to the configured Telegram chat.
func (c *Client) SendMessage(text string) error {
	payload := sendMessageRequest{
		ChatID: c.chatID,
		Text:   text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	url := apiBase + c.botToken + "/sendMessage"
	resp, err := c.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("telegram error: %s", apiResp.Description)
	}
	return nil
}
