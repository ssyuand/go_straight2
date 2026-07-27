package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthCreationAndPasswordVerification(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	auth, password, err := loadOrCreateAuth(path)
	if err != nil {
		t.Fatal(err)
	}
	if password == "" || !auth.passwordMatches("admin", password) {
		t.Fatal("generated credentials do not verify")
	}
	if auth.passwordMatches("admin", password+"wrong") {
		t.Fatal("invalid password was accepted")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("auth file mode=%o, want 600", info.Mode().Perm())
	}
}

func TestAuthMiddlewareAndLogin(t *testing.T) {
	auth, password, err := loadOrCreateAuth(filepath.Join(t.TempDir(), "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	protected := auth.middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("protected"))
	}))

	w := httptest.NewRecorder()
	protected.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://recorder.local/", nil))
	if w.Code != http.StatusSeeOther || !strings.HasPrefix(w.Header().Get("Location"), "/login") {
		t.Fatalf("unauthenticated page status=%d location=%q", w.Code, w.Header().Get("Location"))
	}

	w = httptest.NewRecorder()
	protected.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://recorder.local/api/status", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated API status=%d", w.Code)
	}

	form := "username=admin&password=" + password + "&next=%2F"
	w = httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://recorder.local/login", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	auth.handleLogin(w, r)
	if w.Code != http.StatusSeeOther || len(w.Result().Cookies()) == 0 {
		t.Fatalf("login status=%d cookies=%v", w.Code, w.Result().Cookies())
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "http://recorder.local/", nil)
	r2.AddCookie(w.Result().Cookies()[0])
	protected.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK || w2.Body.String() != "protected" {
		t.Fatalf("authenticated status=%d body=%q", w2.Code, w2.Body.String())
	}
}

func TestSafeNextRejectsExternalRedirect(t *testing.T) {
	if got := safeNext("//evil.example/path"); got != "/" {
		t.Fatalf("unsafe redirect accepted: %q", got)
	}
	if got := safeNext("%2Fapi%2Fstatus"); got != "/api/status" {
		t.Fatalf("safe redirect changed: %q", got)
	}
}
