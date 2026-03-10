package ports

// ICacheCleaner limpa cache do proxy para forçar novo circuito.
// Implementado por adapters na infrastructure (ex.: cache TOR).
type ICacheCleaner interface {
	CleanCache()
}
