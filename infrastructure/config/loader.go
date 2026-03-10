package config

import (
	"fmt"
	"os"
	"strconv"

	"worker/application/dto"
)

// Load lê variáveis de ambiente, aplica defaults e valida. Retorna erro descritivo se inválido.
// Único ponto de leitura de config; main e composição usam apenas o Config retornado.
func Load() (*dto.Config, error) {
	token := os.Getenv("USER_PROXY_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("config: USER_PROXY_TOKEN não definido")
	}
	torPassword := os.Getenv("TOR_CONTROL_PASSWORD")
	if torPassword == "" {
		return nil, fmt.Errorf("config: TOR_CONTROL_PASSWORD não definido (em Docker o entrypoint gera automaticamente)")
	}
	listenPort := envInt("LISTEN_PORT", 8080)
	proxyPort := envInt("PROXY_PORT", 1080)
	autoRefresh := envInt("AUTO_REFRESH_MINUTES", 5)
	if listenPort <= 0 || listenPort > 65535 {
		return nil, fmt.Errorf("config: LISTEN_PORT inválido (%d)", listenPort)
	}
	if proxyPort <= 0 || proxyPort > 65535 {
		return nil, fmt.Errorf("config: PROXY_PORT inválido (%d)", proxyPort)
	}
	if autoRefresh < 0 {
		return nil, fmt.Errorf("config: AUTO_REFRESH_MINUTES não pode ser negativo (%d)", autoRefresh)
	}
	return &dto.Config{
		ProxyToken:         token,
		ListenPort:         listenPort,
		ProxyPort:          proxyPort,
		TorPassword:        torPassword,
		AutoRefreshMinutes: autoRefresh,
	}, nil
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
