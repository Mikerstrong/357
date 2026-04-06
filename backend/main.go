package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"stock357/scraper"
	"stock357/strategy"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/analyze", cors(analyzeHandler))
	mux.HandleFunc("/api/calc",    cors(calcHandler))
	mux.HandleFunc("/api/save",    cors(saveHandler))
	mux.HandleFunc("/api/upload",  cors(uploadHandler))

	log.Printf("Backend running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// analyzeHandler fetches stock data and calculates 3-5-7 strategy metrics using the current market price.
func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	ticker := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("ticker")))
	if ticker == "" {
		writeErr(w, 400, "ticker is required")
		return
	}

	bankroll, err := strconv.ParseFloat(r.URL.Query().Get("bankroll"), 64)
	if err != nil || bankroll <= 0 {
		writeErr(w, 400, "bankroll must be a positive number")
		return
	}

	tradesPerYear := 20
	if tpy := r.URL.Query().Get("trades_per_year"); tpy != "" {
		if v, err := strconv.Atoi(tpy); err == nil && v > 0 {
			tradesPerYear = v
		}
	}

	data, err := scraper.FetchYahoo(ticker)
	if err != nil {
		writeErr(w, 502, err.Error())
		return
	}

	result := strategy.Calculate(data, bankroll, 0, tradesPerYear)
	writeJSON(w, result)
}

// calcHandler fetches ATR for a ticker but prices the trade at a user-supplied entry (e.g. what they paid).
func calcHandler(w http.ResponseWriter, r *http.Request) {
	ticker := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("ticker")))
	if ticker == "" {
		writeErr(w, 400, "ticker is required")
		return
	}

	entry, err := strconv.ParseFloat(r.URL.Query().Get("entry"), 64)
	if err != nil || entry <= 0 {
		writeErr(w, 400, "entry must be a positive price (the price you paid)")
		return
	}

	bankroll, err := strconv.ParseFloat(r.URL.Query().Get("bankroll"), 64)
	if err != nil || bankroll <= 0 {
		writeErr(w, 400, "bankroll must be a positive number")
		return
	}

	tradesPerYear := 20
	if tpy := r.URL.Query().Get("trades_per_year"); tpy != "" {
		if v, err := strconv.Atoi(tpy); err == nil && v > 0 {
			tradesPerYear = v
		}
	}

	data, err := scraper.FetchYahoo(ticker)
	if err != nil {
		writeErr(w, 502, err.Error())
		return
	}

	result := strategy.Calculate(data, bankroll, entry, tradesPerYear)
	writeJSON(w, result)
}

// saveHandler persists the analysis result to a JSON file in ./results/.
func saveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeErr(w, 405, "method not allowed")
		return
	}

	var result strategy.AnalysisResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		writeErr(w, 400, "invalid JSON body")
		return
	}

	if err := os.MkdirAll("results", 0755); err != nil {
		writeErr(w, 500, "failed to create results directory")
		return
	}

	ts := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.json", result.Symbol, ts)
	path := filepath.Join("results", filename)

	f, err := os.Create(path)
	if err != nil {
		writeErr(w, 500, fmt.Sprintf("failed to create file: %v", err))
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		writeErr(w, 500, "failed to write JSON")
		return
	}

	writeJSON(w, map[string]string{"file": path, "status": "saved"})
}

// uploadHandler FTPs the most recently saved results file to the configured FTP server.
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeErr(w, 405, "method not allowed")
		return
	}

	host := os.Getenv("FTP_HOST")
	user := os.Getenv("FTP_USER")
	pass := os.Getenv("FTP_PASS")
	dir := os.Getenv("FTP_DIR")
	if dir == "" {
		dir = "/357"
	}

	if host == "" || user == "" {
		writeErr(w, 503, "FTP not configured — set FTP_HOST, FTP_USER, FTP_PASS in .env")
		return
	}

	// Find the most recently saved file (skip .env files)
	entries, err := os.ReadDir("results")
	if err != nil {
		writeErr(w, 404, "no saved results to upload — save first")
		return
	}
	var latest os.DirEntry
	for i := len(entries) - 1; i >= 0; i-- {
		if !strings.HasSuffix(entries[i].Name(), ".env") {
			latest = entries[i]
			break
		}
	}
	if latest == nil {
		writeErr(w, 404, "no saved results to upload — save first")
		return
	}
	localPath := filepath.Join("results", latest.Name())
	remoteURL := fmt.Sprintf("ftp://%s:%s@%s%s/%s", user, pass, host, dir, latest.Name())

	out, err := exec.Command("curl", "-T", localPath, "--ftp-create-dirs", remoteURL).CombinedOutput()
	if err != nil {
		writeErr(w, 500, fmt.Sprintf("FTP upload failed: %s", string(out)))
		return
	}

	writeJSON(w, map[string]string{
		"status": "uploaded",
		"file":   latest.Name(),
		"remote": fmt.Sprintf("ftp://%s%s/%s", host, dir, latest.Name()),
	})
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
