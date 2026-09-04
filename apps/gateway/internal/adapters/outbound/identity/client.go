package identityclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	clientID   string
	secret     string
	scopes     []string
	httpClient *http.Client
	now        func() time.Time
	mu         sync.Mutex
	token      string
	expiresAt  time.Time
}

func NewClient(baseURL, clientID, secret string, scopes []string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(clientID) == "" || strings.TrimSpace(secret) == "" || len(scopes) == 0 {
		return nil, errors.New("identity service-token configuration is incomplete")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), clientID: clientID, secret: secret, scopes: append([]string(nil), scopes...), httpClient: httpClient, now: time.Now}, nil
}

func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.token != "" && c.now().Before(c.expiresAt.Add(-30*time.Second)) {
		token := c.token
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()
	payload, err := json.Marshal(map[string]any{"client_id": c.clientID, "client_secret": c.secret, "scopes": c.scopes})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/auth/service-token", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("identity service-token status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return "", err
	}
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.AccessToken == "" || result.ExpiresIn <= 0 {
		return "", errors.New("invalid identity service-token response")
	}
	c.mu.Lock()
	c.token = result.AccessToken
	c.expiresAt = c.now().Add(time.Duration(result.ExpiresIn) * time.Second)
	token := c.token
	c.mu.Unlock()
	return token, nil
}

func (c *Client) Refresh(ctx context.Context) (string, error) {
	c.mu.Lock()
	c.token = ""
	c.expiresAt = time.Time{}
	c.mu.Unlock()
	return c.Token(ctx)
}
