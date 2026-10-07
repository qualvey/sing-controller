package probe

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/qualvey/sing-controller/internal/models"
)

func postWebhook(ctx context.Context, delivery models.WebhookDelivery) (int, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		delivery.URL,
		bytes.NewReader(delivery.Payload),
	)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "sing-controller-node-probe/1")
	request.Header.Set("X-Sing-Controller-Event", delivery.EventType)
	request.Header.Set("X-Sing-Controller-Event-ID", delivery.EventID)
	if delivery.Secret != "" {
		request.Header.Set("X-Secret", delivery.Secret)
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		if message != "" {
			return response.StatusCode, fmt.Errorf("webhook returned HTTP %d: %s", response.StatusCode, message)
		}
		return response.StatusCode, fmt.Errorf("webhook returned HTTP %d", response.StatusCode)
	}
	return response.StatusCode, nil
}
