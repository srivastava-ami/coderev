package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestUpsertCommentUpdatesInPlace drives the full find-or-upsert flow against a
// fake GitHub: the first call must POST a new comment, and the second must PATCH
// the SAME comment id rather than creating a duplicate.
func TestUpsertCommentUpdatesInPlace(t *testing.T) {
	const commentID = int64(4242)

	var stored issueComment // server-side state: the one comment we own
	var (
		postCount  int
		patchCount int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("missing/wrong auth header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/issues/7/comments"):
			var list []issueComment
			if stored.ID != 0 {
				list = append(list, stored)
			}
			writeJSON(t, w, list)

		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues/7/comments"):
			postCount++
			stored = issueComment{ID: commentID, Body: readBody(t, r)}
			w.WriteHeader(http.StatusCreated)
			writeJSON(t, w, stored)

		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/issues/comments/"+itoa(commentID)):
			patchCount++
			stored.Body = readBody(t, r)
			writeJSON(t, w, stored)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewWithToken("test-token")
	client.baseURL = srv.URL

	if err := client.UpsertComment("octo/repo", 7, "first body"); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if err := client.UpsertComment("octo/repo", 7, "second body"); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	if postCount != 1 {
		t.Errorf("expected exactly 1 POST (create), got %d", postCount)
	}
	if patchCount != 1 {
		t.Errorf("expected exactly 1 PATCH (update), got %d", patchCount)
	}
	if stored.ID != commentID {
		t.Errorf("comment id changed: got %d want %d", stored.ID, commentID)
	}
	if !strings.Contains(stored.Body, "second body") {
		t.Errorf("comment not updated, body = %q", stored.Body)
	}
	if !strings.Contains(stored.Body, commentMarker) {
		t.Errorf("marker missing from stored body = %q", stored.Body)
	}
}

func TestUpsertCommentRejectsBadRepo(t *testing.T) {
	client := NewWithToken("test-token")
	if err := client.UpsertComment("not-a-slug", 1, "body"); err == nil {
		t.Fatal("expected error for malformed repo, got nil")
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	var payload struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return payload.Body
}

// itoa renders an int64 without pulling strconv into the switch above.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
