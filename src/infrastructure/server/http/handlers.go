package http

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"time"

	"worker/src/application/worker/ports"
)

type ipResponse struct {
	IP  string `json:"ip"`
	TOR bool   `json:"tor"`
}

type healthResponse struct {
	Status        string `json:"status"`
	TOR           string `json:"tor"`
	CircuitIP     string `json:"circuit_ip"`
	Goroutines    int    `json:"goroutines"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Handlers gerencia os handlers HTTP para o servidor HTTP.
type Handlers struct {
	ProxyMgr  ports.IProxyManager
	StartTime time.Time
}

func NewHandlers(proxyMgr ports.IProxyManager, startTime time.Time) *Handlers {
	return &Handlers{ProxyMgr: proxyMgr, StartTime: startTime}
}

func (h *Handlers) HandleGetIP(w http.ResponseWriter, r *http.Request) {
	ip, err := h.ProxyMgr.FetchIP(r.Context())
	if err != nil {
		log.Printf("http: GET /getIp — falha: %v", err)
		writeError(w, http.StatusServiceUnavailable, "tor_unavailable")
		return
	}
	log.Printf("http: GET /getIp — %s", ip)
	writeJSON(w, http.StatusOK, ipResponse{IP: ip, TOR: true})
}

func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	torRunning := h.ProxyMgr.IsRunning()
	ip := h.ProxyMgr.CurrentIP()
	var status, torStatus string
	if torRunning {
		status, torStatus = "ok", "connected"
	} else {
		status, torStatus = "down", "disconnected"
	}
	writeJSON(w, http.StatusOK, healthResponse{
		Status: status, TOR: torStatus, CircuitIP: ip,
		Goroutines: runtime.NumGoroutine(),
		UptimeSeconds: int64(time.Since(h.StartTime).Seconds()),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, errorResponse{Error: code})
}
