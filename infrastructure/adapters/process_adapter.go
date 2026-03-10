package adapters

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"worker/application/worker/ports"
)

const (
	torControlAddr        = "127.0.0.1:9051"
	torBootstrapTimeout   = 120 * time.Second
	torKillTimeout        = 10 * time.Second
	torControlPollInterval = 2 * time.Second
)

// ProcessAdapter controla o processo do daemon TOR; implementa ports.IProcessController.
type ProcessAdapter struct {
	mu     sync.Mutex
	torPID int
}

func NewProcessAdapter() *ProcessAdapter {
	return &ProcessAdapter{}
}

func (p *ProcessAdapter) StartProcess() error {
	cmd := exec.Command("tor", "-f", "/etc/tor/torrc")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("exec tor: %w", err)
	}
	pid := cmd.Process.Pid
	p.mu.Lock()
	p.torPID = pid
	p.mu.Unlock()
	log.Printf("tor: daemon iniciado (PID %d)", pid)
	go func() {
		_ = cmd.Wait()
		p.clearPID(pid)
	}()
	return nil
}

func (p *ProcessAdapter) Stop() error {
	pid, err := p.currentPID()
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("tor: FindProcess(%d): %w", pid, err)
	}
	log.Printf("tor: SIGTERM → PID %d", pid)
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("tor: SIGTERM: %w", err)
	}
	deadline := time.Now().Add(torKillTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		if syscall.Kill(pid, 0) != nil {
			log.Printf("tor: processo %d encerrado", pid)
			p.clearPID(pid)
			return nil
		}
	}
	log.Printf("tor: SIGKILL → PID %d", pid)
	_ = proc.Signal(syscall.SIGKILL)
	time.Sleep(2 * time.Second)
	p.clearPID(pid)
	return nil
}

func (p *ProcessAdapter) WaitReady(ctx context.Context) error {
	log.Printf("tor: aguardando ControlPort %s...", torControlAddr)
	deadline := time.Now().Add(torBootstrapTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := net.DialTimeout("tcp", torControlAddr, 2*time.Second)
		if err == nil {
			conn.Close()
			log.Printf("tor: ControlPort pronta")
			return nil
		}
		time.Sleep(torControlPollInterval)
	}
	return fmt.Errorf("tor: ControlPort não disponível após %v", torBootstrapTimeout)
}

func (p *ProcessAdapter) IsRunning() bool {
	pid, err := p.currentPID()
	if err != nil {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func (p *ProcessAdapter) currentPID() (int, error) {
	p.mu.Lock()
	pid := p.torPID
	p.mu.Unlock()
	if pid > 0 {
		return pid, nil
	}
	out, err := exec.Command("pidof", "tor").Output()
	if err != nil {
		return 0, fmt.Errorf("tor: pidof: %w", err)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return 0, fmt.Errorf("tor: TOR não encontrado")
	}
	pid, err = strconv.Atoi(fields[0])
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func (p *ProcessAdapter) clearPID(pid int) {
	p.mu.Lock()
	if p.torPID == pid {
		p.torPID = 0
	}
	p.mu.Unlock()
}

var _ ports.IProcessController = (*ProcessAdapter)(nil)
