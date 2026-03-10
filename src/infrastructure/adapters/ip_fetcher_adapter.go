package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"worker/src/application/worker/ports"
)

const ipFetchTimeout = 30 * time.Second

var ipCheckURLs = []string{
	"https://api.ipify.org?format=json",
	"https://api64.ipify.org?format=json",
	"https://icanhazip.com",
}

// IPFetcherAdapter guarda o client HTTP (via proxy) e o IP atual; implementa ports.IIPFetcher.
type IPFetcherAdapter struct {
	mu        sync.RWMutex
	client    *http.Client
	currentIP string
}

func NewIPFetcherAdapter(client *http.Client) *IPFetcherAdapter {
	return &IPFetcherAdapter{client: client}
}

func (f *IPFetcherAdapter) FetchIP(ctx context.Context) (string, error) {
	f.mu.RLock()
	c := f.client
	f.mu.RUnlock()
	if c == nil {
		return "", nil
	}
	return fetchIPViaClient(ctx, c)
}

func (f *IPFetcherAdapter) CurrentIP() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.currentIP
}

func (f *IPFetcherAdapter) SetCurrentIP(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.currentIP = ip
}

func fetchIPViaClient(ctx context.Context, client *http.Client) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, ipFetchTimeout)
	defer cancel()
	var lastErr error
	for _, url := range ipCheckURLs {
		ip, err := fetchIPFromURL(ctx, client, url)
		if err == nil && ip != "" {
			return ip, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("ip: todos falharam: %w", lastErr)
}

func fetchIPFromURL(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512))
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(body))
	if strings.HasPrefix(url, "https://icanhazip.com") {
		if s == "" {
			return "", fmt.Errorf("resposta vazia")
		}
		return s, nil
	}
	var parsed struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.IP == "" {
		return "", fmt.Errorf("campo ip ausente")
	}
	return parsed.IP, nil
}

var _ ports.IIPFetcher = (*IPFetcherAdapter)(nil)
