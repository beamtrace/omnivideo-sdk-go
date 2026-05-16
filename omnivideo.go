// Package omnivideo is a Go SDK for the Omni Video API
// (https://omnivideo.net/) — generate video and image content with the
// Gemini Omni Video series of models.
//
// Sign in at https://omnivideo.net/ to issue a sk-... API key.
package omnivideo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const DefaultBaseURL = "https://omnivideo.net/api/v1"

// TaskStatus values returned by the Omni Video API.
type TaskStatus int

const (
	TaskStatusQueued  TaskStatus = 1
	TaskStatusRunning TaskStatus = 2
	TaskStatusSuccess TaskStatus = 3
	TaskStatusFailed  TaskStatus = 4
)

// Task represents the state of a generation job.
type Task struct {
	TaskID     string         `json:"task_id"`
	TaskStatus TaskStatus     `json:"task_status"`
	ImageURL   string         `json:"image_url,omitempty"`
	VideoURL   string         `json:"video_url,omitempty"`
	Credits    int            `json:"credits,omitempty"`
	Msg        string         `json:"msg,omitempty"`
	Raw        map[string]any `json:"-"`
}

// Done reports whether the task has reached a terminal state.
func (t Task) Done() bool {
	return t.TaskStatus == TaskStatusSuccess || t.TaskStatus == TaskStatusFailed
}

// OutputURL returns video_url if present, else image_url.
func (t Task) OutputURL() string {
	if t.VideoURL != "" {
		return t.VideoURL
	}
	return t.ImageURL
}

// CreateTaskInput is the request body for POST /tasks/create.
type CreateTaskInput struct {
	ModelID     string   `json:"model_id"`
	Prompt      string   `json:"prompt"`
	ImageURLs   []string `json:"image_urls,omitempty"`
	AspectRatio string   `json:"aspect_ratio,omitempty"`
}

// Error is the SDK error type. Code is the business code (200=ok, 0=biz fail).
type Error struct {
	Message string
	Code    int
	Status  int
}

func (e *Error) Error() string { return e.Message }

// Client talks to the Omni Video API.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

// NewClient constructs a Client. If apiKey is empty it reads OMNIVIDEO_API_KEY.
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		apiKey = os.Getenv("OMNIVIDEO_API_KEY")
	}
	if apiKey == "" {
		return nil, &Error{Message: "missing API key — set OMNIVIDEO_API_KEY or call NewClient with a key. Get one at https://omnivideo.net/"}
	}
	return &Client{
		APIKey:  apiKey,
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// CreateTask submits a job and returns the task with task_id set.
func (c *Client) CreateTask(ctx context.Context, in CreateTaskInput) (*Task, error) {
	payload, err := c.do(ctx, http.MethodPost, "/tasks/create", in)
	if err != nil {
		return nil, err
	}
	return payloadToTask(payload), nil
}

// GetTask fetches the current state of a task.
func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	payload, err := c.do(ctx, http.MethodGet, "/tasks/"+taskID, nil)
	if err != nil {
		return nil, err
	}
	return payloadToTask(payload), nil
}

// RunOptions configures Run.
type RunOptions struct {
	PollInterval time.Duration
	MaxWait      time.Duration
}

// Run creates a task and polls until it reaches a terminal state.
func (c *Client) Run(ctx context.Context, in CreateTaskInput, opts RunOptions) (*Task, error) {
	if opts.PollInterval <= 0 {
		opts.PollInterval = 3 * time.Second
	}
	if opts.MaxWait <= 0 {
		opts.MaxWait = 10 * time.Minute
	}
	task, err := c.CreateTask(ctx, in)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(opts.MaxWait)
	for !task.Done() {
		if time.Now().After(deadline) {
			return task, &Error{Message: fmt.Sprintf("task %s did not finish within %s", task.TaskID, opts.MaxWait)}
		}
		select {
		case <-ctx.Done():
			return task, ctx.Err()
		case <-time.After(opts.PollInterval):
		}
		task, err = c.GetTask(ctx, task.TaskID)
		if err != nil {
			return nil, err
		}
	}
	if task.TaskStatus == TaskStatusFailed {
		msg := task.Msg
		if msg == "" {
			msg = "task failed"
		}
		return task, &Error{Message: msg, Code: int(TaskStatusFailed)}
	}
	return task, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any) (map[string]any, error) {
	url := strings.TrimRight(c.BaseURL, "/") + path
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &Error{Message: "unauthorized — check your OMNIVIDEO_API_KEY (https://omnivideo.net/)", Status: 401}
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, &Error{Message: fmt.Sprintf("invalid JSON: %s", truncate(string(data), 200)), Status: resp.StatusCode}
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &Error{Message: stringOr(payload["msg"], fmt.Sprintf("HTTP %d", resp.StatusCode)), Status: resp.StatusCode, Code: intOr(payload["code"], 0)}
	}
	if code, ok := payload["code"].(float64); ok && code != 200 {
		return nil, &Error{Message: stringOr(payload["msg"], "business error"), Code: int(code)}
	}
	return payload, nil
}

func payloadToTask(p map[string]any) *Task {
	t := &Task{Raw: p}
	if v, ok := p["task_id"].(string); ok {
		t.TaskID = v
	}
	if v, ok := p["task_status"].(float64); ok {
		t.TaskStatus = TaskStatus(int(v))
	} else {
		t.TaskStatus = TaskStatusQueued
	}
	if v, ok := p["image_url"].(string); ok {
		t.ImageURL = v
	}
	if v, ok := p["video_url"].(string); ok {
		t.VideoURL = v
	}
	if v, ok := p["credits"].(float64); ok {
		t.Credits = int(v)
	}
	if v, ok := p["msg"].(string); ok {
		t.Msg = v
	}
	return t
}

func stringOr(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

func intOr(v any, fallback int) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return fallback
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Compile-time check that *Error satisfies error.
var _ error = (*Error)(nil)

// ErrUnauthorized is returned when the API rejects the key.
var ErrUnauthorized = errors.New("unauthorized")
