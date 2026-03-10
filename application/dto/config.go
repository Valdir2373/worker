package dto

// Config contém todos os parâmetros de configuração do Worker (saída do config.Load).
// O orquestrador identifica este worker pela URL (domínio) da conexão; não há WORKER_ID.
type Config struct {
	ProxyToken         string // USER_PROXY_TOKEN
	ListenPort         int
	ProxyPort          int
	TorPassword        string
	AutoRefreshMinutes int
}
