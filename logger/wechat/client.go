// Package wechat 提供微信服务号报警客户端。
package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultAPIBaseURL = "https://api.weixin.qq.com"
	httpTimeout       = 10 * time.Second
	tokenRefreshAhead = 5 * time.Minute
	alertDataMaxRunes = 100
)

// Config 是微信服务号凭证配置。
type Config struct {
	AppID      string `json:"appId" yaml:"appId"`
	AppSecret  string `json:"appSecret" yaml:"appSecret"`
	ToUser     string `json:"toUser" yaml:"toUser"`
	TemplateID string `json:"templateId" yaml:"templateId"`
}

// Client 是微信服务号 API 客户端，负责模板消息与 access_token 管理。
type Client struct {
	appID       string
	appSecret   string
	httpClient  *http.Client
	apiBaseURL  string
	tokenMu     sync.Mutex
	accessToken string
	tokenExpiry time.Time

	toUser     string
	templateID string
}

// NewClient 创建微信服务号客户端。
func NewClient(conf *Config) *Client {
	return &Client{
		appID:      conf.AppID,
		appSecret:  conf.AppSecret,
		httpClient: &http.Client{Timeout: httpTimeout},
		apiBaseURL: defaultAPIBaseURL,
		toUser:     conf.ToUser,
		templateID: conf.TemplateID,
	}
}

// SendAlert sends one alert through the configured template. Queueing, rate
// limiting, and duplicate coalescing are handled by AlertLogger.
func (s *Client) SendAlert(level, message string, count int) error {
	title := level
	if count > 1 {
		title = fmt.Sprintf("%s（%d次）", title, count)
	}

	_, err := s.SendTemplateMessage(context.Background(), &TemplateMessage{
		ToUser:     s.toUser,
		TemplateID: s.templateID,
		Data: TemplateData{
			"title":   {Value: title},
			"content": {Value: truncateAlertData(message)},
		},
	})
	return err
}

func truncateAlertData(message string) string {
	runes := []rune(message)
	if len(runes) <= alertDataMaxRunes {
		return message
	}
	return string(runes[:alertDataMaxRunes])
}

// TemplateMessage 是 /cgi-bin/message/template/send 接口的请求参数。
type TemplateMessage struct {
	ToUser      string       `json:"touser"`
	TemplateID  string       `json:"template_id"`
	URL         string       `json:"url,omitempty"`
	MiniProgram *MiniProgram `json:"miniprogram,omitempty"`
	Data        TemplateData `json:"data"`
}

// MiniProgram 配置可选的跳转小程序。
type MiniProgram struct {
	AppID    string `json:"appid"`
	PagePath string `json:"pagepath,omitempty"`
}

// TemplateData 将模板字段名（例如 first、keyword1）映射为显示内容。
type TemplateData map[string]TemplateDataValue

// TemplateDataValue 是单个模板字段的内容。
type TemplateDataValue struct {
	Value string `json:"value"`
	Color string `json:"color,omitempty"`
}

// TemplateMessageSendResult 是微信受理模板消息后返回的结果。
type TemplateMessageSendResult struct {
	MessageID int64 `json:"msgid"`
}

// APIError 是微信 API 返回的业务错误。
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("weixin API error: %d - %s", e.Code, e.Message)
}

// SendTemplateMessage 发送服务号模板消息。微信返回 access_token 失效时，会清空本地
// 缓存并使用新 token 重试一次。
func (s *Client) SendTemplateMessage(ctx context.Context, message *TemplateMessage) (*TemplateMessageSendResult, error) {
	if message == nil || message.ToUser == "" || message.TemplateID == "" || message.Data == nil {
		return nil, fmt.Errorf("template message touser, template_id and data are required")
	}

	for attempt := 0; attempt < 2; attempt++ {
		accessToken, err := s.GetAccessToken(ctx)
		if err != nil {
			return nil, err
		}

		body, err := json.Marshal(message)
		if err != nil {
			return nil, fmt.Errorf("encode template message: %w", err)
		}
		endpoint := s.apiBaseURL + "/cgi-bin/message/template/send"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"?"+url.Values{"access_token": {accessToken}}.Encode(), bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create template message request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send template message: %w", err)
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read template message response: %w", readErr)
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("send template message: unexpected HTTP status %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
		}

		var result struct {
			MessageID int64  `json:"msgid"`
			ErrorCode int    `json:"errcode"`
			ErrorMsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return nil, fmt.Errorf("decode template message response: %w", err)
		}
		if result.ErrorCode == 0 {
			return &TemplateMessageSendResult{MessageID: result.MessageID}, nil
		}

		apiErr := &APIError{Code: result.ErrorCode, Message: result.ErrorMsg}
		if attempt == 0 && (apiErr.Code == 40001 || apiErr.Code == 40014 || apiErr.Code == 42001) {
			s.invalidateAccessToken(accessToken)
			continue
		}
		return nil, apiErr
	}

	return nil, fmt.Errorf("send template message: access token retry exhausted")
}

// GetAccessToken 返回有效的服务号 access_token。token 缓存在当前服务实例内，并在
// 到期前预留 tokenRefreshAhead 的刷新时间。
func (s *Client) GetAccessToken(ctx context.Context) (string, error) {
	if s.appID == "" || s.appSecret == "" {
		return "", fmt.Errorf("weixin appId and appSecret must be configured")
	}

	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return s.accessToken, nil
	}

	endpoint := s.apiBaseURL + "/cgi-bin/token"
	query := url.Values{
		"grant_type": {"client_credential"},
		"appid":      {s.appID},
		"secret":     {s.appSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("create access token request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get access token: %w", err)
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if readErr != nil {
		return "", fmt.Errorf("read access token response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("get access token: unexpected HTTP status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrorCode   int    `json:"errcode"`
		ErrorMsg    string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode access token response: %w", err)
	}
	if result.ErrorCode != 0 {
		return "", &APIError{Code: result.ErrorCode, Message: result.ErrorMsg}
	}
	if result.AccessToken == "" || result.ExpiresIn <= 0 {
		return "", fmt.Errorf("get access token: invalid response")
	}

	ttl := time.Duration(result.ExpiresIn)*time.Second - tokenRefreshAhead
	if ttl <= 0 {
		ttl = time.Second
	}
	s.accessToken = result.AccessToken
	s.tokenExpiry = time.Now().Add(ttl)
	return result.AccessToken, nil
}

func (s *Client) invalidateAccessToken(token string) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	if s.accessToken == token {
		s.accessToken = ""
		s.tokenExpiry = time.Time{}
	}
}
