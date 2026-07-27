package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type ConfigUpdateResponse struct {
	Status          string `json:"status"`
	RestartRequired bool   `json:"restart_required"`
	Message         string `json:"message"`
}

func (a *App) configFilePath() string {
	if a.ConfigPath != "" {
		return a.ConfigPath
	}
	return "config.json"
}

func validateTargetURLs(values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("target_urls 至少需要一個頻道")
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		value = strings.TrimSpace(value)
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || !strings.Contains(value, "@") {
			return fmt.Errorf("target_urls[%d] 必須是包含 @ 的有效 HTTP/HTTPS 網址", index)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("target_urls[%d] 重複", index)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func saveConfigAtomically(path string, cfg Config) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".config-*.json")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0644); err != nil {
		_ = temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func readConfigFile(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

func (a *App) handleAPIConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		cfg, err := readConfigFile(a.configFilePath())
		if err != nil {
			http.Error(w, `{"error":"無法讀取設定檔"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(cfg)
		return
	}
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var next Config
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&next); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"JSON 設定格式錯誤: %s"}`, strings.ReplaceAll(err.Error(), `"`, `'`)), http.StatusBadRequest)
		return
	}
	for index := range next.TargetURLs {
		next.TargetURLs[index] = strings.TrimSpace(next.TargetURLs[index])
	}
	if err := validateConfig(next); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, strings.ReplaceAll(err.Error(), `"`, `'`)), http.StatusBadRequest)
		return
	}
	current, err := readConfigFile(a.configFilePath())
	if err != nil {
		http.Error(w, `{"error":"無法讀取目前設定"}`, http.StatusInternalServerError)
		return
	}
	restartRequired := current.WebPort != next.WebPort || !reflect.DeepEqual(current.TargetURLs, next.TargetURLs)
	if err := saveConfigAtomically(a.configFilePath(), next); err != nil {
		log.Printf("[CONFIG-ERROR] Web 設定儲存失敗: %v", err)
		http.Error(w, `{"error":"設定檔儲存失敗"}`, http.StatusInternalServerError)
		return
	}

	// Apply schedule-only values immediately. Port and channel topology are
	// intentionally activated by the existing restart flow.
	a.configMu.Lock()
	a.Config.ProbeStart = next.ProbeStart
	a.Config.ProbeEnd = next.ProbeEnd
	a.Config.ProbeInterval = next.ProbeInterval
	a.Config.ProbeSleepDeep = next.ProbeSleepDeep
	a.configMu.Unlock()
	a.StreamsMu.RLock()
	for _, stream := range a.Streams {
		select {
		case stream.ReloadChan <- struct{}{}:
		default:
		}
	}
	a.StreamsMu.RUnlock()
	log.Printf("[CONFIG] Web 設定已儲存 restart_required=%t", restartRequired)
	_ = json.NewEncoder(w).Encode(ConfigUpdateResponse{Status: "saved", RestartRequired: restartRequired, Message: "設定已儲存"})
}
