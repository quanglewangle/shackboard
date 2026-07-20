package main

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/quanglewangle/shackboard/internal/cluster"
	"github.com/quanglewangle/shackboard/internal/spacewx"
)

// buildHash is set at build time via -ldflags "-X main.buildHash=...".
var buildHash = "dev"

//go:embed web
var webFS embed.FS

func main() {
	port := getenv("SHACKBOARD_PORT", "8093")
	clusterHost := getenv("SHACKBOARD_CLUSTER_HOST", "dxc.ve7cc.net:23")
	clusterCall := os.Getenv("SHACKBOARD_CLUSTER_CALL")
	if clusterCall == "" {
		log.Fatal("SHACKBOARD_CLUSTER_CALL is required (the callsign to log into the DX cluster with)")
	}
	spacewxURL := getenv("SHACKBOARD_SPACEWX_URL", "https://www.hamqsl.com/solarxml.php")
	bufSize := getenvInt("SHACKBOARD_SPOT_BUFFER_SIZE", 200)
	maxAge := getenvDuration("SHACKBOARD_SPOT_MAX_AGE", 2*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wxCache := spacewx.NewCache()
	go spacewx.Poll(ctx, wxCache, http.DefaultClient, spacewxURL, time.Hour)

	spotBuf := cluster.NewBuffer(bufSize, maxAge)
	clusterClient := cluster.NewClient(clusterHost, clusterCall, spotBuf)
	go clusterClient.Run(ctx)

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("embed web fs: %v", err)
	}
	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	indexHTML = []byte(strings.ReplaceAll(string(indexHTML), "{{BUILD}}", buildHash))

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/spacewx", func(w http.ResponseWriter, r *http.Request) {
		data, _ := wxCache.Get()
		if data == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "space weather data not yet available",
			})
			return
		}
		writeJSON(w, http.StatusOK, data)
	})

	mux.HandleFunc("GET /api/spots", func(w http.ResponseWriter, r *http.Request) {
		limit := 100
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if limit > bufSize {
			limit = bufSize
		}
		spots := spotBuf.Recent(limit)
		writeJSON(w, http.StatusOK, map[string]any{
			"spots":             spots,
			"count":             len(spots),
			"cluster_connected": clusterClient.Connected(),
			"cluster_host":      clusterHost,
		})
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_, wxErr := wxCache.Get()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":            "ok",
			"build":             buildHash,
			"cluster_connected": clusterClient.Connected(),
			"spacewx_ok":        wxErr == nil,
		})
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexHTML)
			return
		}
		http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
	})

	addr := ":" + port
	log.Printf("shackboard listening on %s (build %s, cluster %s)", addr, buildHash, clusterHost)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
