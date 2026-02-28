package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TelegramClient 提供簡單的 sendMessage API 封裝。
type TelegramClient struct {
	token      string
	chatID     int64
	prefix     string
	baseURL    string
	httpClient *http.Client
}

func NewTelegramClient(token string, chatID int64, prefix string) *TelegramClient {
	return &TelegramClient{
		token:   token,
		chatID:  chatID,
		prefix:  prefix,
		baseURL: "https://api.telegram.org",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendMessage 將文字訊息推送到指定 chat。
func (c *TelegramClient) SendMessage(ctx context.Context, text string) error {
	if c == nil {
		return fmt.Errorf("telegram client is nil")
	}
	if c.token == "" || c.chatID == 0 {
		return fmt.Errorf("telegram token or chat_id missing")
	}

	fullText := text
	if c.prefix != "" {
		fullText = fmt.Sprintf("[%s] %s", c.prefix, text)
	}

	payload := map[string]interface{}{
		"chat_id": c.chatID,
		"text":    fullText,
	}
	body, _ := json.Marshal(payload)

	resp, err := c.httpClient.Post(fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, c.token), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram send failed status=%d body=%s", resp.StatusCode, string(raw))
	}
	return nil
}
