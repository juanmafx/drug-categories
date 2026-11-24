package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/juanmafx/drug-categories/backend/internal/graphqlapi"
	"github.com/juanmafx/drug-categories/backend/internal/store"
)

const (
	testOrigin   = "http://localhost:5173"
	testToken    = "0123456789abcdef0123456789abcdef"
	listBody     = `{"query":"query { categories { id name description } }"}`
	mutationBody = `{"query":"mutation($name:String!){ createCategory(name:$name){ id name } }","variables":{"name":"Antivirals"}}`
)

func newTestHandler(t *testing.T, token string) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service, err := graphqlapi.NewService(store.NewMemory(store.DefaultSeed()...), logger)
	if err != nil {
		t.Fatal(err)
	}
	return NewHandler(service, Options{AllowedOrigins: []string{testOrigin}, WriteToken: token, Logger: logger})
}

func post(handler http.Handler, body string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestListCategoriesEndToEnd(t *testing.T) {
	recorder := post(newTestHandler(t, ""), listBody, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body)
	}
	var payload struct {
		Data struct {
			Categories []struct {
				Name string `json:"name"`
			} `json:"categories"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data.Categories) != 1 || payload.Data.Categories[0].Name != "Analgesics" {
		t.Fatalf("got %+v", payload)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	recorder := post(newTestHandler(t, ""), listBody, nil)
	for _, header := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "Cache-Control"} {
		if recorder.Header().Get(header) == "" {
			t.Fatalf("missing %s", header)
		}
	}
}

func TestOriginPolicy(t *testing.T) {
	handler := newTestHandler(t, "")

	allowed := post(handler, listBody, map[string]string{"Origin": testOrigin})
	if allowed.Code != http.StatusOK || allowed.Header().Get("Access-Control-Allow-Origin") != testOrigin {
		t.Fatalf("status %d, header %q", allowed.Code, allowed.Header().Get("Access-Control-Allow-Origin"))
	}

	denied := post(handler, mutationBody, map[string]string{"Origin": "https://attacker.example"})
	if denied.Code != http.StatusForbidden || denied.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("status %d", denied.Code)
	}
	listed := post(handler, listBody, nil)
	if strings.Contains(listed.Body.String(), "Antivirals") {
		t.Fatal("cross-origin mutation was applied")
	}
}

func TestPreflight(t *testing.T) {
	handler := newTestHandler(t, "")
	request := httptest.NewRequest(http.MethodOptions, "/graphql", nil)
	request.Header.Set("Origin", testOrigin)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || !strings.Contains(recorder.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("status %d", recorder.Code)
	}
}

func TestRejectsNonPost(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/graphql?query={categories{id}}", nil)
	recorder := httptest.NewRecorder()
	newTestHandler(t, "").ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", recorder.Code)
	}
}

func TestRejectsSimpleRequestContentTypes(t *testing.T) {
	handler := newTestHandler(t, "")
	for _, contentType := range []string{"text/plain", "application/x-www-form-urlencoded", "multipart/form-data; boundary=x", ""} {
		recorder := post(handler, mutationBody, map[string]string{"Content-Type": contentType})
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("%q: status %d", contentType, recorder.Code)
		}
	}
}

func TestRequestValidation(t *testing.T) {
	handler := newTestHandler(t, "")
	oversizedQuery, _ := json.Marshal(map[string]string{"query": "{" + strings.Repeat(" ", maxQueryBytes) + "categories{id}}"})
	cases := map[string]struct {
		body   string
		status int
	}{
		"malformed json": {`{`, http.StatusBadRequest},
		"trailing json":  {listBody + `{}`, http.StatusBadRequest},
		"missing query":  {`{}`, http.StatusBadRequest},
		"oversized body": {`{"query":"` + strings.Repeat("a", maxBodyBytes) + `"}`, http.StatusRequestEntityTooLarge},
		"long query":     {string(oversizedQuery), http.StatusRequestEntityTooLarge},
		"invalid query":  {`{"query":"{ nope }"}`, http.StatusBadRequest},
	}
	for label, testCase := range cases {
		t.Run(label, func(t *testing.T) {
			recorder := post(handler, testCase.body, nil)
			if recorder.Code != testCase.status {
				t.Fatalf("status %d, want %d: %s", recorder.Code, testCase.status, recorder.Body)
			}
		})
	}
}

func TestWriteTokenEnforcement(t *testing.T) {
	handler := newTestHandler(t, testToken)

	if recorder := post(handler, listBody, nil); recorder.Code != http.StatusOK {
		t.Fatalf("reads must stay public, status %d", recorder.Code)
	}
	for _, header := range []string{"", "Bearer wrong", "Basic " + testToken, testToken} {
		headers := map[string]string{}
		if header != "" {
			headers["Authorization"] = header
		}
		recorder := post(handler, mutationBody, headers)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("header %q: status %d", header, recorder.Code)
		}
	}
	recorder := post(handler, mutationBody, map[string]string{"Authorization": "Bearer " + testToken})
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Antivirals") {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body)
	}
}

func TestRateLimitReturns429(t *testing.T) {
	handler := newTestHandler(t, "")
	for range requestsPerMin {
		if recorder := post(handler, listBody, nil); recorder.Code != http.StatusOK {
			t.Fatalf("status %d", recorder.Code)
		}
	}
	recorder := post(handler, listBody, nil)
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") == "" {
		t.Fatalf("status %d", recorder.Code)
	}
}

func TestRateLimiterWindowAndIsolation(t *testing.T) {
	current := time.Unix(1000, 0)
	limiter := newRateLimiter(1, time.Minute, func() time.Time { return current })

	if !limiter.allow("192.0.2.1:1000") || limiter.allow("192.0.2.1:2000") {
		t.Fatal("per-client limit not applied")
	}
	if !limiter.allow("192.0.2.2:1000") {
		t.Fatal("clients must be isolated")
	}
	if !limiter.allow("[2001:db8:1:1::1]:1000") || limiter.allow("[2001:db8:1:1::2]:1000") {
		t.Fatal("IPv6 clients must share a /64 bucket")
	}
	current = current.Add(time.Minute)
	if !limiter.allow("192.0.2.1:1000") {
		t.Fatal("window did not reset")
	}
}

func TestRateLimiterCapsTrackedClients(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute, time.Now)
	for index := range maxTrackedClients {
		address := [4]byte{10, byte(index >> 16), byte(index >> 8), byte(index)}
		remote := strings.Join([]string{itoa(address[0]), itoa(address[1]), itoa(address[2]), itoa(address[3])}, ".") + ":1"
		if !limiter.allow(remote) {
			t.Fatalf("client %d denied", index)
		}
	}
	if limiter.allow("198.51.100.1:1") {
		t.Fatal("expected denial once the client table is full")
	}
}

func itoa(value byte) string {
	encoded, _ := json.Marshal(int(value))
	return string(encoded)
}

func TestHealthEndpoint(t *testing.T) {
	handler := newTestHandler(t, "")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status %d", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", recorder.Code)
	}
}
