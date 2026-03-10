package domain

type Command struct {
	Command string `json:"command"`
}

type Event struct {
	Event         string `json:"event"`
	WorkerID      string `json:"worker_id"`
	IP            string `json:"ip,omitempty"`
	Socks5Port    int    `json:"socks5_port,omitempty"`
	ListenPort    int    `json:"listen_port,omitempty"`
	Message       string `json:"message,omitempty"`
	UptimeSeconds int64  `json:"uptime_seconds,omitempty"`
	Goroutines    int    `json:"goroutines,omitempty"`
}
