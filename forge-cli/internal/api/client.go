// Package api is the typed HTTP client for forge-platform.
//
// Public surface is intentionally narrow: one Client, one method per
// endpoint. No generated code, no transparent retries, no caching.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrAuthorizationPending = errors.New("authorization pending")
var ErrExpiredToken = errors.New("device code expired")

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) request(ctx context.Context, method, path string, body any, out any) error {
	var buf io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, buf)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return apiError(resp.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func apiError(status int, body []byte) error {
	var oauthErr struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &oauthErr) == nil {
		switch oauthErr.Error {
		case "authorization_pending":
			return ErrAuthorizationPending
		case "expired_token":
			return ErrExpiredToken
		}
	}
	return fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(string(body)))
}

// ---- Device-code (RFC 8628) ----------------------------------------------

type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

func (c *Client) RequestDeviceCode(ctx context.Context) (DeviceCode, error) {
	var out DeviceCode
	err := c.request(ctx, http.MethodPost, "/oauth/device/code", nil, &out)
	return out, err
}

type DeviceToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Handle      string `json:"handle"`
	ID          string `json:"id"`
}

func (c *Client) PollDeviceToken(ctx context.Context, deviceCode string) (DeviceToken, error) {
	var out DeviceToken
	err := c.request(ctx, http.MethodPost, "/oauth/device/token",
		map[string]string{"device_code": deviceCode}, &out)
	return out, err
}

// ---- Me ------------------------------------------------------------------

type Me struct {
	ID          string `json:"id"`
	Handle      string `json:"handle"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Provider    string `json:"provider"`
}

func (c *Client) Me(ctx context.Context) (Me, error) {
	var out Me
	err := c.request(ctx, http.MethodGet, "/me", nil, &out)
	return out, err
}
