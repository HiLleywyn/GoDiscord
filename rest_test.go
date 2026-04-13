package discord

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers - custom RoundTripper that records the request and returns a
// canned response, so we can drive the real RestClient methods end-to-end
// without touching discord.com.
// ---------------------------------------------------------------------------

type recordingTransport struct {
	lastReq  *http.Request
	lastBody []byte
	respBody string
	respCode int
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.lastReq = req
	if req.Body != nil {
		t.lastBody, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	code := t.respCode
	if code == 0 {
		code = http.StatusOK
	}
	body := t.respBody
	if body == "" {
		body = "[]"
	}
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func newTestRest(rt *recordingTransport) *RestClient {
	return &RestClient{token: "test", client: &http.Client{Transport: rt}}
}

// ---------------------------------------------------------------------------
// Input validation - BulkDeleteMessages
// ---------------------------------------------------------------------------

// TestBulkDeleteMessages_Validation verifies the in-process guard before any
// HTTP call is made.
func TestBulkDeleteMessages_Validation(t *testing.T) {
	r := &RestClient{token: "x", client: nil}

	// Too few.
	if err := r.BulkDeleteMessages("chan", []string{"only-one"}); err == nil {
		t.Error("BulkDeleteMessages with 1 ID should return an error")
	}

	// Empty slice.
	if err := r.BulkDeleteMessages("chan", []string{}); err == nil {
		t.Error("BulkDeleteMessages with 0 IDs should return an error")
	}

	// 101 IDs is above the documented Discord maximum (100), so the local
	// validator must reject it before any HTTP call is made.
	ids := make([]string, 101)
	if err := r.BulkDeleteMessages("chan", ids); err == nil {
		t.Error("BulkDeleteMessages with 101 IDs should return an error")
	}

	// Exactly 100 IDs is within the documented Discord maximum. The local
	// validator must not reject it; the actual request goes through the
	// recording transport so we don't touch discord.com.
	rt := &recordingTransport{respCode: http.StatusNoContent}
	rOK := newTestRest(rt)
	hundred := make([]string, 100)
	for i := range hundred {
		hundred[i] = "id"
	}
	if err := rOK.BulkDeleteMessages("chan", hundred); err != nil {
		t.Errorf("BulkDeleteMessages with 100 IDs returned err: %v", err)
	}
	if rt.lastReq == nil {
		t.Fatal("expected an HTTP request for 100 IDs")
	}
	if !strings.HasSuffix(rt.lastReq.URL.Path, "/channels/chan/messages/bulk-delete") {
		t.Errorf("unexpected request path: %s", rt.lastReq.URL.Path)
	}
}

// ---------------------------------------------------------------------------
// GetMessages clamping - exercises the real method, not a re-implementation
// ---------------------------------------------------------------------------

func TestGetMessages_LimitClamping(t *testing.T) {
	cases := []struct {
		input int
		want  string // expected ?limit= value
	}{
		{0, "limit=1"},
		{-5, "limit=1"},
		{50, "limit=50"},
		{100, "limit=100"},
		{200, "limit=100"},
	}
	for _, tc := range cases {
		rt := &recordingTransport{respBody: "[]"}
		r := newTestRest(rt)
		if _, err := r.GetMessages("chan", tc.input); err != nil {
			t.Errorf("GetMessages(%d) error: %v", tc.input, err)
			continue
		}
		if rt.lastReq == nil {
			t.Errorf("GetMessages(%d): no request observed", tc.input)
			continue
		}
		query := rt.lastReq.URL.RawQuery
		if query != tc.want {
			t.Errorf("GetMessages(%d) sent query %q, want %q", tc.input, query, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// BanMember clamping - exercises the real method via the recorded body
// ---------------------------------------------------------------------------

func TestBanMember_DaysClamping(t *testing.T) {
	cases := []struct {
		input int
		want  int
	}{
		{-1, 0},
		{0, 0},
		{3, 3},
		{7, 7},
		{8, 7},
		{99, 7},
	}
	for _, tc := range cases {
		rt := &recordingTransport{respCode: http.StatusNoContent}
		r := newTestRest(rt)
		if err := r.BanMember("guild", "user", tc.input); err != nil {
			t.Errorf("BanMember(%d) error: %v", tc.input, err)
			continue
		}
		if len(rt.lastBody) == 0 {
			t.Errorf("BanMember(%d): no body observed", tc.input)
			continue
		}
		var payload map[string]int
		if err := json.Unmarshal(rt.lastBody, &payload); err != nil {
			t.Errorf("BanMember(%d): body unmarshal: %v (body=%q)", tc.input, err, string(rt.lastBody))
			continue
		}
		if payload["delete_message_days"] != tc.want {
			t.Errorf("BanMember(%d) sent delete_message_days=%d, want %d", tc.input, payload["delete_message_days"], tc.want)
		}
	}
}
