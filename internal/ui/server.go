package ui

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/catu-ai/easyharness/internal/planui"
	"github.com/catu-ai/easyharness/internal/reviewui"
	"github.com/catu-ai/easyharness/internal/status"
	"github.com/catu-ai/easyharness/internal/timeline"
)

//go:embed static
var embeddedStatic embed.FS

const productDisplayName = "easyharness"

type Server struct {
	Workdir     string
	Host        string
	Port        int
	Stdout      io.Writer
	Stderr      io.Writer
	OpenBrowser bool
}

func (s Server) Run(ctx context.Context) error {
	host := strings.TrimSpace(s.Host)
	if host == "" {
		host = "127.0.0.1"
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(s.Port)))
	if err != nil {
		return fmt.Errorf("listen for harness ui: %w", err)
	}
	defer listener.Close()

	url := "http://" + listener.Addr().String()
	if s.Stdout != nil {
		_, _ = fmt.Fprintf(s.Stdout, "Harness UI listening at %s\n", url)
	}

	if s.OpenBrowser {
		if err := openBrowser(url); err != nil && s.Stderr != nil {
			_, _ = fmt.Fprintf(s.Stderr, "open browser: %v\n", err)
		}
	}

	handler, err := NewHandler(s.Workdir)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve harness ui: %w", err)
	}
	return nil
}

func NewHandler(workdir string) (http.Handler, error) {
	staticFS, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		return nil, fmt.Errorf("load embedded ui assets: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeStatusJSON(w, status.Service{Workdir: workdir}.Read())
	})
	mux.HandleFunc("/api/plan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writePlanJSON(w, planui.Service{Workdir: workdir}.Read())
	})
	mux.HandleFunc("/api/timeline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeTimelineJSON(w, timeline.Service{Workdir: workdir}.Read())
	})
	mux.HandleFunc("/api/review", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeReviewJSON(w, reviewui.Service{Workdir: workdir}.Read())
	})
	mux.Handle("/", spaHandler(staticFS, workdir))
	return mux, nil
}

func spaHandler(staticFS fs.FS, workdir string) http.Handler {
	files := http.FileServer(http.FS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/api" {
			http.NotFound(w, r)
			return
		}

		requestPath := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		switch requestPath {
		case ".", "":
			serveIndex(staticFS, workdir, w)
			return
		}

		if entry, err := fs.Stat(staticFS, requestPath); err == nil && !entry.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		serveIndex(staticFS, workdir, w)
	})
}

func serveIndex(staticFS fs.FS, workdir string, w http.ResponseWriter) {
	data, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		http.Error(w, "missing embedded ui index", http.StatusInternalServerError)
		return
	}
	workdirJSON, err := json.Marshal(filepath.Clean(workdir))
	if err != nil {
		http.Error(w, "encode workdir", http.StatusInternalServerError)
		return
	}
	repoNameJSON, err := json.Marshal(filepath.Base(filepath.Clean(workdir)))
	if err != nil {
		http.Error(w, "encode repo name", http.StatusInternalServerError)
		return
	}
	productNameJSON, err := json.Marshal(productDisplayName)
	if err != nil {
		http.Error(w, "encode product name", http.StatusInternalServerError)
		return
	}
	page := strings.ReplaceAll(string(data), "\"__HARNESS_UI_WORKDIR__\"", string(workdirJSON))
	page = strings.ReplaceAll(page, "\"__HARNESS_UI_REPO_NAME__\"", string(repoNameJSON))
	page = strings.ReplaceAll(page, "\"__HARNESS_UI_PRODUCT_NAME__\"", string(productNameJSON))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, page)
}

func writeStatusJSON(w http.ResponseWriter, result status.Result) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func writePlanJSON(w http.ResponseWriter, result planui.Result) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func writeTimelineJSON(w http.ResponseWriter, result timeline.Result) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func writeReviewJSON(w http.ResponseWriter, result reviewui.Result) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
