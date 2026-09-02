package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var ErrNotModified = errors.New("subscription not modified (304)")

// Fetcher HTTP 订阅拉取器
type Fetcher struct {
	client *http.Client
}

func NewFetcher(timeout time.Duration) *Fetcher {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Fetch 拉取远程订阅内容，支持 ETag 差量校验
func (f *Fetcher) Fetch(ctx context.Context, urlStr string, lastETag string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置伪装 User-Agent 与 Accept 标头，确保机场服务端正确下发节点
	req.Header.Set("User-Agent", "sing-box/1.14.0 clash.meta")
	req.Header.Set("Accept", "application/json, text/yaml, application/x-yaml, text/plain, */*")

	if lastETag != "" {
		req.Header.Set("If-None-Match", lastETag)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("请求订阅链接失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, lastETag, ErrNotModified
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("订阅服务器返回错误 HTTP %s", resp.Status)
	}

	newETag := resp.Header.Get("ETag")
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 限制最大 10MB
	if err != nil {
		return nil, "", fmt.Errorf("读取订阅内容失败: %w", err)
	}

	return body, newETag, nil
}
