package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"worker/infrastructure/config"
	"worker/infrastructure/modules"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("worker: iniciando")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("worker: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := modules.NewWorkerApp(cfg)
	if err := app.Run(ctx); err != nil {
		log.Fatalf("worker: %v", err)
	}
	log.Println("worker: encerrado")
}
