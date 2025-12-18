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
	httpClient *http.Client
}

func NewTelegramClient(token string, chatID int64) *TelegramClient {
	return &TelegramClient{
		token:  token,
		chatID: chatID,
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

	payload := map[string]interface{}{
		"chat_id": c.chatID,
		"text":    text,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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
