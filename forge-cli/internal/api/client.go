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

// ---- Repos ---------------------------------------------------------------

type Repo struct {
	ID          string `json:"id"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	CreatedAt   string `json:"created_at"`
	CloneURL    string `json:"clone_url"`
}

type CreateRepoInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
	InitReadme  bool   `json:"init_readme,omitempty"`
	ImportURL   string `json:"import_url,omitempty"`
}

type UpdateRepoInput struct {
	Description *string `json:"description,omitempty"`
	Visibility  *string `json:"visibility,omitempty"`
	Name        *string `json:"name,omitempty"`
}

func (c *Client) ListRepos(ctx context.Context) ([]Repo, error) {
	var out []Repo
	err := c.request(ctx, http.MethodGet, "/repos", nil, &out)
	return out, err
}

func (c *Client) GetRepo(ctx context.Context, owner, name string) (Repo, error) {
	var out Repo
	err := c.request(ctx, http.MethodGet, "/repos/"+owner+"/"+name, nil, &out)
	return out, err
}

func (c *Client) CreateRepo(ctx context.Context, in CreateRepoInput) (Repo, error) {
	var out Repo
	err := c.request(ctx, http.MethodPost, "/repos", in, &out)
	return out, err
}

func (c *Client) UpdateRepo(ctx context.Context, owner, name string, in UpdateRepoInput) (Repo, error) {
	var out Repo
	err := c.request(ctx, http.MethodPatch, "/repos/"+owner+"/"+name, in, &out)
	return out, err
}

func (c *Client) DeleteRepo(ctx context.Context, owner, name string) error {
	return c.request(ctx, http.MethodDelete, "/repos/"+owner+"/"+name, nil, nil)
}

// ---- Issues --------------------------------------------------------------

type Assignee struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Handle string `json:"handle"`
}

type Issue struct {
	ID        string    `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	Author    string    `json:"author"`
	Assignee  *Assignee `json:"assignee,omitempty"`
	CreatedAt string    `json:"created_at"`
	ClosedAt  string    `json:"closed_at,omitempty"`
}

type IssueComment struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

type IssueDetail struct {
	Issue    Issue          `json:"issue"`
	Comments []IssueComment `json:"comments"`
}

type CreateIssueInput struct {
	Title          string `json:"title"`
	Body           string `json:"body,omitempty"`
	AssigneeUserID string `json:"assignee_user_id,omitempty"`
}

func (c *Client) ListIssues(ctx context.Context, owner, name, state string) ([]Issue, error) {
	path := "/repos/" + owner + "/" + name + "/issues"
	if state != "" && state != "all" {
		path += "?state=" + state
	}
	var out []Issue
	err := c.request(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) GetIssue(ctx context.Context, owner, name string, number int) (IssueDetail, error) {
	var out IssueDetail
	err := c.request(ctx, http.MethodGet,
		fmt.Sprintf("/repos/%s/%s/issues/%d", owner, name, number), nil, &out)
	return out, err
}

func (c *Client) CreateIssue(ctx context.Context, owner, name string, in CreateIssueInput) (Issue, error) {
	var out Issue
	err := c.request(ctx, http.MethodPost,
		"/repos/"+owner+"/"+name+"/issues", in, &out)
	return out, err
}

func (c *Client) CommentOnIssue(ctx context.Context, owner, name string, number int, body string) (IssueComment, error) {
	var out IssueComment
	err := c.request(ctx, http.MethodPost,
		fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, name, number),
		map[string]string{"body": body}, &out)
	return out, err
}

func (c *Client) CloseIssue(ctx context.Context, owner, name string, number int) (Issue, error) {
	var out Issue
	err := c.request(ctx, http.MethodPost,
		fmt.Sprintf("/repos/%s/%s/issues/%d/close", owner, name, number), nil, &out)
	return out, err
}

func (c *Client) ReopenIssue(ctx context.Context, owner, name string, number int) (Issue, error) {
	var out Issue
	err := c.request(ctx, http.MethodPost,
		fmt.Sprintf("/repos/%s/%s/issues/%d/reopen", owner, name, number), nil, &out)
	return out, err
}

type Run struct {
	ID              string `json:"id"`
	State           string `json:"state"`
	CancelRequested bool   `json:"cancel_requested,omitempty"`
	SandboxID       string `json:"sandbox_id,omitempty"`
	ErrorCategory   string `json:"error_category,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	StartedAt       string `json:"started_at,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
	PRNumber        string `json:"pr_number,omitempty"`
	StreamURL       string `json:"stream_url,omitempty"`
}

type DashboardRun struct {
	ID          string `json:"id"`
	State       string `json:"state"`
	IssueNumber int    `json:"issue_number"`
	IssueTitle  string `json:"issue_title"`
	RepoOwner   string `json:"repo_owner"`
	RepoName    string `json:"repo_name"`
	PRNumber    *int   `json:"pr_number,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func (c *Client) ListMyRuns(ctx context.Context) ([]DashboardRun, error) {
	var out []DashboardRun
	err := c.request(ctx, http.MethodGet, "/me/runs", nil, &out)
	return out, err
}

func (c *Client) GetRun(ctx context.Context, id string) (Run, error) {
	var out Run
	err := c.request(ctx, http.MethodGet, "/runs/"+id, nil, &out)
	return out, err
}

func (c *Client) CancelRun(ctx context.Context, id string) error {
	return c.request(ctx, http.MethodPost, "/runs/"+id+"/cancel", nil, nil)
}

func (c *Client) AssignAgent(ctx context.Context, owner, name string, number int) (Run, error) {
	var out Run
	err := c.request(ctx, http.MethodPost,
		fmt.Sprintf("/repos/%s/%s/issues/%d/assign-agent", owner, name, number), nil, &out)
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
