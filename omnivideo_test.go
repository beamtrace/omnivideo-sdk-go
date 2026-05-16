package omnivideo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type step struct {
	body map[string]any
	code int
}

func newTestServer(t *testing.T, steps ...step) *httptest.Server {
	t.Helper()
	i := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if i >= len(steps) {
			t.Fatalf("unexpected extra request: %s %s", r.Method, r.URL.Path)
		}
		s := steps[i]
		i++
		w.Header().Set("Content-Type", "application/json")
		if s.code != 0 {
			w.WriteHeader(s.code)
		}
		_ = json.NewEncoder(w).Encode(s.body)
	}))
}

func TestCreateTaskReturnsTaskID(t *testing.T) {
	srv := newTestServer(t, step{body: map[string]any{"code": 200, "task_id": "abc", "task_status": 1}})
	defer srv.Close()
	c := &Client{APIKey: "sk-test", BaseURL: srv.URL, HTTP: srv.Client()}
	task, err := c.CreateTask(context.Background(), CreateTaskInput{ModelID: "gpt-image-2", Prompt: "x"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if task.TaskID != "abc" || task.TaskStatus != TaskStatusQueued {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestMissingAPIKey(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestRunPollsToSuccess(t *testing.T) {
	srv := newTestServer(t,
		step{body: map[string]any{"code": 200, "task_id": "t1", "task_status": 1}},
		step{body: map[string]any{"code": 200, "task_id": "t1", "task_status": 2}},
		step{body: map[string]any{"code": 200, "task_id": "t1", "task_status": 3, "image_url": "https://x/y.png"}},
	)
	defer srv.Close()
	c := &Client{APIKey: "sk-test", BaseURL: srv.URL, HTTP: srv.Client()}
	task, err := c.Run(context.Background(), CreateTaskInput{ModelID: "gpt-image-2", Prompt: "x"}, RunOptions{PollInterval: time.Millisecond, MaxWait: 5 * time.Second})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if task.TaskStatus != TaskStatusSuccess || task.OutputURL() != "https://x/y.png" {
		t.Fatalf("unexpected: %+v", task)
	}
}

func TestBusinessErrorRaised(t *testing.T) {
	srv := newTestServer(t, step{body: map[string]any{"code": 0, "msg": "insufficient credits"}})
	defer srv.Close()
	c := &Client{APIKey: "sk-test", BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.CreateTask(context.Background(), CreateTaskInput{ModelID: "gpt-image-2", Prompt: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	e, ok := err.(*Error)
	if !ok || e.Message != "insufficient credits" {
		t.Fatalf("unexpected error: %v", err)
	}
}
