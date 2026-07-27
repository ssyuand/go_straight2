package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// appLogWriter turns the existing human-oriented log messages into consistent,
// grep-friendly logfmt without forcing every call site to be rewritten at once.
type appLogWriter struct {
	mu  sync.Mutex
	out io.Writer
}

func configureLogging() {
	log.SetFlags(0)
	output := io.Writer(os.Stderr)
	if os.Getenv("LIVETOOL_LAUNCHD") == "1" {
		if rotating, err := newRotatingFileWriter("livetool.log", logMaxBytes(), 7); err == nil {
			output = rotating
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "logging setup failed: %v\n", err)
		}
	}
	log.SetOutput(&appLogWriter{out: output})
}

func logMaxBytes() int64 {
	const defaultMax = int64(50 * 1024 * 1024)
	value := strings.TrimSpace(os.Getenv("LIVETOOL_LOG_MAX_MB"))
	mb, err := strconv.ParseInt(value, 10, 64)
	if err != nil || mb < 1 {
		return defaultMax
	}
	return mb * 1024 * 1024
}

type rotatingFileWriter struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	backups  int
	file     *os.File
	size     int64
}

func newRotatingFileWriter(path string, maxBytes int64, backups int) (*rotatingFileWriter, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &rotatingFileWriter{path: path, maxBytes: maxBytes, backups: backups, file: file, size: stat.Size()}, nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size > 0 && w.size+int64(len(p)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingFileWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	if w.backups > 0 {
		if err := os.Remove(fmt.Sprintf("%s.%d", w.path, w.backups)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for index := w.backups - 1; index >= 1; index-- {
		oldPath := fmt.Sprintf("%s.%d", w.path, index)
		newPath := fmt.Sprintf("%s.%d", w.path, index+1)
		if err := os.Rename(oldPath, newPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if w.backups > 0 {
		if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	w.file = file
	w.size = 0
	return nil
}

func (w *appLogWriter) Write(p []byte) (int, error) {
	originalLen := len(p)
	message := strings.TrimSpace(stripANSI(string(p)))
	if message == "" {
		return originalLen, nil
	}

	component, message := splitLogComponent(message)
	level := inferLogLevel(component, message)
	line := fmt.Sprintf("ts=%s level=%s component=%s msg=%s\n",
		time.Now().Format(time.RFC3339Nano), level, quoteLogValue(component), quoteLogValue(message))

	w.mu.Lock()
	_, err := io.WriteString(w.out, line)
	w.mu.Unlock()
	return originalLen, err
}

func splitLogComponent(message string) (string, string) {
	if strings.HasPrefix(message, "[") {
		if end := strings.IndexByte(message, ']'); end > 1 {
			return strings.TrimSpace(message[1:end]), strings.TrimSpace(message[end+1:])
		}
	}
	return "APP", message
}

func inferLogLevel(component, message string) string {
	upper := strings.ToUpper(component + " " + message)
	switch {
	case strings.Contains(upper, "ERROR"), strings.Contains(upper, "崩潰"), strings.Contains(upper, "失敗"):
		return "ERROR"
	case strings.Contains(upper, "WARNING"), strings.Contains(upper, "WARN"), strings.Contains(upper, "警告"), strings.Contains(upper, "非法"):
		return "WARN"
	default:
		return "INFO"
	}
}

func quoteLogValue(value string) string {
	if value != "" && !strings.ContainsAny(value, " \t\r\n=\"") {
		return value
	}
	return strconv.Quote(value)
}

func stripANSI(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); {
		if value[i] == 0x1b && i+1 < len(value) && value[i+1] == '[' {
			i += 2
			for i < len(value) {
				c := value[i]
				i++
				if c >= '@' && c <= '~' {
					break
				}
			}
			continue
		}
		b.WriteByte(value[i])
		i++
	}
	return b.String()
}

type responseStatusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseStatusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseStatusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func (w *responseStatusWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

var requestSequence atomic.Uint64

func requestDiagnostics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := fmt.Sprintf("%x-%06x", started.Unix(), requestSequence.Add(1))
		w.Header().Set("X-Request-ID", requestID)
		recorder := &responseStatusWriter{ResponseWriter: w}

		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("[HTTP-ERROR] panic recovered request_id=%s method=%s path=%s panic=%q stack=%q",
					requestID, r.Method, r.URL.Path, fmt.Sprint(recovered), string(debug.Stack()))
				if recorder.status == 0 {
					http.Error(recorder, "Internal Server Error", http.StatusInternalServerError)
				}
			}
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			duration := time.Since(started)
			// The dashboard polls these endpoints every two seconds. Successful fast
			// polls add noise but no diagnostic value; failures and slow polls remain.
			if status < http.StatusBadRequest && duration < time.Second &&
				(r.URL.Path == "/api/status" || r.URL.Path == "/api/logs") {
				return
			}
			component := "HTTP"
			if status >= http.StatusInternalServerError {
				component = "HTTP-ERROR"
			} else if status >= http.StatusBadRequest {
				component = "HTTP-WARNING"
			}
			log.Printf("[%s] request completed request_id=%s method=%s path=%q query=%q status=%d bytes=%d duration_ms=%.3f remote_ip=%s user_agent=%q",
				component,
				requestID, r.Method, r.URL.Path, r.URL.RawQuery, status, recorder.bytes,
				float64(duration.Microseconds())/1000, clientIP(r), r.UserAgent())
		}()

		next.ServeHTTP(recorder, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
