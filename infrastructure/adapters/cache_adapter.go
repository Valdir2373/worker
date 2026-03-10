package adapters

import (
	"log"
	"os"

	"worker/application/worker/ports"
)

var torCacheFiles = []string{
	"cached-certs", "cached-microdesc-consensus", "cached-microdescs",
	"cached-microdescs.new", "state", "diff-cache", "control_auth_cookie",
}

// CacheAdapter implements ports.ICacheCleaner (TOR cache).
type CacheAdapter struct{}

func NewCacheAdapter() *CacheAdapter {
	return &CacheAdapter{}
}

func (c *CacheAdapter) CleanCache() {
	for _, name := range torCacheFiles {
		path := "/var/lib/tor/" + name
		if err := os.RemoveAll(path); err != nil {
			log.Printf("tor: aviso %s: %v", path, err)
		} else {
			log.Printf("tor: cache removido: %s", path)
		}
	}
}

var _ ports.ICacheCleaner = (*CacheAdapter)(nil)
