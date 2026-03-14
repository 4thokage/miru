package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type TestServer struct {
	Server   *httptest.Server
	Client   *http.Client
	BaseURL  string
	Requests []*http.Request
}

func NewTestServer(handler http.Handler) *TestServer {
	ts := httptest.NewServer(handler)
	return &TestServer{
		Server:   ts,
		Client:   ts.Client(),
		BaseURL:  ts.URL,
		Requests: make([]*http.Request, 0),
	}
}

func (ts *TestServer) Close() {
	ts.Server.Close()
}

func (ts *TestServer) MustGet(t *testing.T, path string) *http.Response {
	resp, err := ts.Client.Get(ts.BaseURL + path)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	return resp
}

func (ts *TestServer) MustPost(t *testing.T, path string, body interface{}) *http.Response {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal body: %v", err)
	}

	resp, err := ts.Client.Post(ts.BaseURL+path, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	return resp
}

func AssertStatus(t *testing.T, got, expected int) {
	if got != expected {
		t.Errorf("Expected status %d, got %d", expected, got)
	}
}

func AssertJSONField(t *testing.T, body []byte, field string, expected interface{}) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if got := result[field]; got != expected {
		t.Errorf("Expected field %s = %v, got %v", field, expected, got)
	}
}

func AssertSuccess(t *testing.T, body []byte) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		t.Errorf("Expected success=true, got %v", result)
	}
}

func AssertError(t *testing.T, body []byte, expectedError string) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	errorMsg, ok := result["error"].(string)
	if !ok || errorMsg != expectedError {
		t.Errorf("Expected error %q, got %v", expectedError, result["error"])
	}
}

type MockResponse struct {
	Body       interface{}
	StatusCode int
	Headers    map[string]string
}

func WithMockServer(t *testing.T, mockResp *MockResponse, testFunc func(ts *TestServer)) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range mockResp.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(mockResp.StatusCode)
		if mockResp.Body != nil {
			json.NewEncoder(w).Encode(mockResp.Body)
		}
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	testFunc(ts)
}

type ContextKey string

const (
	ContextKeyUserID ContextKey = "user_id"
	ContextKeyStart  ContextKey = "start_time"
)

func WithTimeout(ctx context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, duration)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func ParseAPIResponse(t *testing.T, body []byte) *APIResponse {
	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Failed to parse API response: %v", err)
	}
	return &resp
}

func ReadBody(t *testing.T, resp *http.Response) []byte {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	return body
}

func NewRequestID() string {
	return time.Now().Format("20060102150405.000000")
}

type CacheEntry struct {
	Key        string
	Value      string
	Expiration time.Duration
	SetAt      time.Time
}

type MockCache struct {
	Entries map[string]*CacheEntry
}

func NewMockCache() *MockCache {
	return &MockCache{
		Entries: make(map[string]*CacheEntry),
	}
}

func (c *MockCache) Get(key string) (string, bool) {
	entry, exists := c.Entries[key]
	if !exists {
		return "", false
	}
	if time.Since(entry.SetAt) > entry.Expiration {
		delete(c.Entries, key)
		return "", false
	}
	return entry.Value, true
}

func (c *MockCache) Set(key, value string, ttl time.Duration) {
	c.Entries[key] = &CacheEntry{
		Key:        key,
		Value:      value,
		Expiration: ttl,
		SetAt:      time.Now(),
	}
}

func (c *MockCache) Delete(key string) {
	delete(c.Entries, key)
}

func (c *MockCache) Exists(key string) bool {
	_, exists := c.Entries[key]
	return exists
}
