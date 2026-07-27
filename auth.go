package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const authHashRounds = 200000

type authFile struct {
	Username      string `json:"username"`
	Salt          string `json:"salt"`
	PasswordHash  string `json:"password_hash"`
	SessionSecret string `json:"session_secret"`
}

type loginAttempt struct {
	Failures     int
	BlockedUntil time.Time
}

type authManager struct {
	config   authFile
	secret   []byte
	mu       sync.Mutex
	attempts map[string]loginAttempt
}

func randomToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func passwordDigest(password, salt string) string {
	digest := sha256.Sum256([]byte(salt + "\x00" + password))
	for i := 1; i < authHashRounds; i++ {
		next := sha256.New()
		_, _ = next.Write(digest[:])
		_, _ = next.Write([]byte(salt))
		copy(digest[:], next.Sum(nil))
	}
	return hex.EncodeToString(digest[:])
}

func loadOrCreateAuth(path string) (*authManager, string, error) {
	data, err := os.ReadFile(path)
	generatedPassword := ""
	var config authFile
	if err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, "", fmt.Errorf("解析認證檔失敗: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("讀取認證檔失敗: %w", err)
	} else {
		salt, saltErr := randomToken(24)
		password, passwordErr := randomToken(15)
		secret, secretErr := randomToken(32)
		if saltErr != nil || passwordErr != nil || secretErr != nil {
			return nil, "", fmt.Errorf("產生安全認證資料失敗")
		}
		config = authFile{Username: "admin", Salt: salt, PasswordHash: passwordDigest(password, salt), SessionSecret: secret}
		encoded, marshalErr := json.MarshalIndent(config, "", "  ")
		if marshalErr != nil {
			return nil, "", marshalErr
		}
		if writeErr := os.WriteFile(path, append(encoded, '\n'), 0600); writeErr != nil {
			return nil, "", fmt.Errorf("建立認證檔失敗: %w", writeErr)
		}
		generatedPassword = password
	}
	if config.Username == "" || config.Salt == "" || config.PasswordHash == "" || config.SessionSecret == "" {
		return nil, "", fmt.Errorf("認證檔缺少必要欄位")
	}
	if err := os.Chmod(path, 0600); err != nil {
		return nil, "", fmt.Errorf("設定認證檔權限失敗: %w", err)
	}
	secret, err := base64.RawURLEncoding.DecodeString(config.SessionSecret)
	if err != nil || len(secret) < 32 {
		return nil, "", fmt.Errorf("session_secret 無效")
	}
	return &authManager{config: config, secret: secret, attempts: make(map[string]loginAttempt)}, generatedPassword, nil
}

func (a *authManager) passwordMatches(username, password string) bool {
	usernameOK := subtle.ConstantTimeCompare([]byte(username), []byte(a.config.Username))
	digest := passwordDigest(password, a.config.Salt)
	hashOK := subtle.ConstantTimeCompare([]byte(digest), []byte(a.config.PasswordHash))
	return usernameOK&hashOK == 1
}

func (a *authManager) newSessionCookie() *http.Cookie {
	expires := time.Now().Add(24 * time.Hour)
	payload := a.config.Username + "|" + strconv.FormatInt(expires.Unix(), 10)
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(payload))
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return &http.Cookie{Name: "livetool_session", Value: value, Path: "/", Expires: expires, MaxAge: 86400, HttpOnly: true, SameSite: http.SameSiteStrictMode}
}

func (a *authManager) validSession(r *http.Request) bool {
	cookie, err := r.Cookie("livetool_session")
	if err != nil {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return false
	}
	payloadBytes, err1 := base64.RawURLEncoding.DecodeString(parts[0])
	signature, err2 := base64.RawURLEncoding.DecodeString(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write(payloadBytes)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return false
	}
	payloadParts := strings.Split(string(payloadBytes), "|")
	if len(payloadParts) != 2 || payloadParts[0] != a.config.Username {
		return false
	}
	expires, err := strconv.ParseInt(payloadParts[1], 10, 64)
	return err == nil && time.Now().Unix() < expires
}

func (a *authManager) loginAllowed(ip string) (bool, time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	attempt := a.attempts[ip]
	if time.Now().Before(attempt.BlockedUntil) {
		return false, time.Until(attempt.BlockedUntil)
	}
	return true, 0
}

func (a *authManager) recordLogin(ip string, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if success {
		delete(a.attempts, ip)
		return
	}
	attempt := a.attempts[ip]
	attempt.Failures++
	if attempt.Failures >= 5 {
		attempt.BlockedUntil = time.Now().Add(30 * time.Second)
		attempt.Failures = 0
	}
	a.attempts[ip] = attempt
}

func safeNext(raw string) string {
	decoded, err := url.QueryUnescape(raw)
	if err != nil || !strings.HasPrefix(decoded, "/") || strings.HasPrefix(decoded, "//") {
		return "/"
	}
	return decoded
}

func (a *authManager) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" || r.URL.Path == "/logout" {
			next.ServeHTTP(w, r)
			return
		}
		if a.validSession(r) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"authentication required"}`))
			return
		}
		http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
	})
}

func (a *authManager) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderLogin(w, safeNext(r.URL.Query().Get("next")), "")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := clientIP(r)
	if allowed, remaining := a.loginAllowed(ip); !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(max(1, int(remaining.Seconds()))))
		renderLogin(w, safeNext(r.FormValue("next")), "登入嘗試過多，請稍後再試")
		return
	}
	success := a.passwordMatches(r.FormValue("username"), r.FormValue("password"))
	a.recordLogin(ip, success)
	if !success {
		log.Printf("[AUTH-WARNING] login failed remote=%s", ip)
		w.WriteHeader(http.StatusUnauthorized)
		renderLogin(w, safeNext(r.FormValue("next")), "帳號或密碼錯誤")
		return
	}
	http.SetCookie(w, a.newSessionCookie())
	log.Printf("[AUTH] login succeeded remote=%s", ip)
	http.Redirect(w, r, safeNext(r.FormValue("next")), http.StatusSeeOther)
}

func (a *authManager) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "livetool_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func renderLogin(w http.ResponseWriter, next, message string) {
	const page = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>LiveTool Login</title><style>
html,body{height:100%}body{margin:0;display:grid;place-items:center;background:radial-gradient(circle at 50% 20%,#25304a,#11111b 48%,#090910);color:#cdd6f4;font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.login{width:min(88vw,360px);padding:28px;border:1px solid rgba(166,227,161,.22);border-radius:16px;background:rgba(24,24,37,.92);box-shadow:0 24px 80px rgba(0,0,0,.5),0 0 35px rgba(166,227,161,.07)}h1{margin:0;color:#a6e3a1;font-size:22px;letter-spacing:.05em}.sub{margin:7px 0 22px;color:#7f849c;font-size:11px}.field{display:block;margin-top:14px;color:#9399b2;font-size:9px;letter-spacing:.1em}.field input{box-sizing:border-box;width:100%;margin-top:6px;padding:12px;border:1px solid #45475a;border-radius:8px;background:#11111b;color:#cdd6f4;font:14px inherit;outline:none}.field input:focus{border-color:#a6e3a1;box-shadow:0 0 0 3px rgba(166,227,161,.08)}button{width:100%;margin-top:20px;padding:12px;border:1px solid rgba(166,227,161,.45);border-radius:8px;background:rgba(166,227,161,.12);color:#a6e3a1;font-weight:900;cursor:pointer}.error{margin:14px 0 0;padding:9px;border-radius:7px;background:rgba(243,139,168,.1);color:#f38ba8;font-size:11px}</style></head><body><form class="login" method="post" action="/login"><h1>LIVE TOOL</h1><div class="sub">AUTHORIZED ACCESS ONLY</div>{{if .Message}}<div class="error">{{.Message}}</div>{{end}}<input type="hidden" name="next" value="{{.Next}}"><label class="field">USERNAME<input name="username" autocomplete="username" required autofocus></label><label class="field">PASSWORD<input name="password" type="password" autocomplete="current-password" required></label><button type="submit">SIGN IN</button></form></body></html>`
	tmpl := template.Must(template.New("login").Parse(page))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, struct{ Next, Message string }{next, message})
}

func authPath(base string) string {
	return filepath.Join(base, "auth.json")
}
