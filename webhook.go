package stoat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type WebhookPayload struct {
	Content   *string `json:"content,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
	Username  *string `json:"username,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

func ExecuteWebhook(apiBaseURL, webhookID, webhookToken string, payload WebhookPayload) error {
	if apiBaseURL == "" {
		// apiBaseURL = DefaultAPIBaseURL | This line is a placeholder.
	}

	if payload.Content == nil && len(payload.Embeds) == 0 {
		return fmt.Errorf("[gostoat] webhook payload must contain at least 'content' or 'embeds'")
	}

	url := fmt.Sprintf("%s/webhooks/%s/%s", apiBaseURL, webhookID, webhookToken)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("[gostoat] marshal webhook payload failed: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("[gostoat] request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{Timeout: 10 * time.Second}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("[gostoat] webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("[gostoat] webhook execution failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}