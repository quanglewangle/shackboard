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

	"github.com/quanglewangle/shackboard/internal/adif"
	"github.com/quanglewangle/shackboard/internal/cluster"
	"github.com/quanglewangle/shackboard/internal/parkspots"
	"github.com/quanglewangle/shackboard/internal/qrzlogbook"
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
	potaURL := getenv("SHACKBOARD_POTA_URL", "https://api.pota.app/spot/activator")
	sotaURL := getenv("SHACKBOARD_SOTA_URL", "https://api2.sota.org.uk/api/spots/100/all")
	qrzLogbookURL := getenv("SHACKBOARD_QRZ_LOGBOOK_URL", "https://logbook.qrz.com/api")
	qrzLogbookKey := os.Getenv("SHACKBOARD_QRZ_LOGBOOK_KEY") // unset = worked-before sync disabled

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wxCache := spacewx.NewCache()
	go spacewx.Poll(ctx, wxCache, http.DefaultClient, spacewxURL, time.Hour)

	spotBuf := cluster.NewBuffer(bufSize, maxAge)
	clusterClient := cluster.NewClient(clusterHost, clusterCall, spotBuf)
	go clusterClient.Run(ctx)

	parkCache := parkspots.NewCache()
	go parkspots.Poll(ctx, parkCache, http.DefaultClient, potaURL, sotaURL, 2*time.Minute)

	logIndex := adif.NewIndex()
	go qrzlogbook.Poll(ctx, logIndex, http.DefaultClient, qrzLogbookURL, qrzLogbookKey, time.Hour)

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
			"spots":             decorateSpots(spots, logIndex),
			"count":             len(spots),
			"cluster_connected": clusterClient.Connected(),
			"cluster_host":      clusterHost,
		})
	})

	mux.HandleFunc("GET /api/park-spots", func(w http.ResponseWriter, r *http.Request) {
		data := parkCache.Get()
		writeJSON(w, http.StatusOK, map[string]any{
			"spots":      decorateParkSpots(data.Spots, logIndex),
			"count":      len(data.Spots),
			"pota_ok":    data.POTAOk,
			"sota_ok":    data.SOTAOk,
			"fetched_at": data.FetchedAt,
		})
	})

	mux.HandleFunc("GET /api/log", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, logIndex.Status())
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

// decoratedSpot/decoratedParkSpot exist so cluster and parkspots never need
// to import adif themselves — main.go is the only place that knows about
// both a spot source and the worked-before index.
type decoratedSpot struct {
	cluster.Spot
	WorkedAny  bool `json:"worked_any"`
	WorkedBand bool `json:"worked_band"`
}

func decorateSpots(spots []cluster.Spot, idx *adif.Index) []decoratedSpot {
	out := make([]decoratedSpot, len(spots))
	for i, s := range spots {
		out[i] = decoratedSpot{
			Spot:       s,
			WorkedAny:  idx.WorkedAny(s.DXCall),
			WorkedBand: idx.WorkedBand(s.DXCall, s.Band),
		}
	}
	return out
}

type decoratedParkSpot struct {
	parkspots.Spot
	WorkedAny  bool `json:"worked_any"`
	WorkedBand bool `json:"worked_band"`
}

func decorateParkSpots(spots []parkspots.Spot, idx *adif.Index) []decoratedParkSpot {
	out := make([]decoratedParkSpot, len(spots))
	for i, s := range spots {
		out[i] = decoratedParkSpot{
			Spot:       s,
			WorkedAny:  idx.WorkedAny(s.Activator),
			WorkedBand: idx.WorkedBand(s.Activator, s.Band),
		}
	}
	return out
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
