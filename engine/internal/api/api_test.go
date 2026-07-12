package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
	"github.com/TheGuyDangerous/Syncy/engine/internal/metadata"
	"github.com/TheGuyDangerous/Syncy/engine/internal/syncengine"
)

const testToken = "test-secret-token"

func newTestServer(t *testing.T) *Server {
	t.Helper()
	id, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	store, err := metadata.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return New(syncengine.New(id, store), testToken)
}

func do(t *testing.T, s *Server, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func TestRequiresToken(t *testing.T) {
	s := newTestServer(t)
	if rec := do(t, s, "GET", "/status", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: code = %d, want 401", rec.Code)
	}
	if rec := do(t, s, "GET", "/status", "", "wrong"); rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: code = %d, want 401", rec.Code)
	}
}

func TestStatus(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "GET", "/status", "", testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	var status statusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if status.DeviceID == "" {
		t.Error("status should include a device id")
	}
}

func TestFolderLifecycle(t *testing.T) {
	s := newTestServer(t)

	rec := do(t, s, "POST", "/folders", `{"id":"photos","path":"/data/photos"}`, testToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /folders code = %d, want 201 (%s)", rec.Code, rec.Body)
	}

	rec = do(t, s, "GET", "/folders", "", testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /folders code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "photos") {
		t.Errorf("folder list missing new folder: %s", rec.Body)
	}

	rec = do(t, s, "DELETE", "/folders/photos", "", testToken)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE code = %d, want 204", rec.Code)
	}

	rec = do(t, s, "DELETE", "/folders/photos", "", testToken)
	if rec.Code != http.StatusNotFound {
		t.Errorf("deleting a missing folder code = %d, want 404", rec.Code)
	}
}

func TestAddFolderInvalidJSON(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "POST", "/folders", `not json`, testToken)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestDevicesAndConflicts(t *testing.T) {
	s := newTestServer(t)
	if rec := do(t, s, "GET", "/devices", "", testToken); rec.Code != http.StatusOK {
		t.Errorf("GET /devices code = %d", rec.Code)
	}
	if rec := do(t, s, "GET", "/conflicts", "", testToken); rec.Code != http.StatusOK {
		t.Errorf("GET /conflicts code = %d", rec.Code)
	}
}

func TestGenerateTokenIsRandom(t *testing.T) {
	a, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	b, _ := GenerateToken()
	if a == b || len(a) != 64 {
		t.Errorf("tokens should be distinct 64-char hex; got %q and %q", a, b)
	}
}
