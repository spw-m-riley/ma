package dashboard

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const sessionFileName = "session.json"

type Command struct {
	stdout io.Writer
}

type Session struct {
	Address   string    `json:"address"`
	StartedAt time.Time `json:"startedAt"`
}

func NewCommand(stdout io.Writer) Command {
	return Command{stdout: stdout}
}

func (Command) Name() string {
	return "dashboard"
}

func (c Command) Run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return c.run(ctx, args)
}

func (c Command) run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	listenAddr := fs.String("listen", "127.0.0.1:0", "listen address for the local dashboard")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: ma dashboard [--listen 127.0.0.1:0]")
	}

	root, err := DefaultRoot()
	if err != nil {
		return err
	}
	store, err := OpenStore(root)
	if err != nil {
		return err
	}

	listener, err := listenLoopback(*listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	sessionPath := filepath.Join(root, sessionFileName)
	if err := writeSession(sessionPath, Session{
		Address:   listener.Addr().String(),
		StartedAt: time.Now().UTC(),
	}); err != nil {
		return err
	}
	defer os.Remove(sessionPath)

	server := &http.Server{
		Handler:           NewServer(store).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if c.stdout != nil {
		fmt.Fprintf(c.stdout, "Dashboard available at http://%s\n", listener.Addr().String())
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-errCh; err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func listenLoopback(addr string) (net.Listener, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid listen address %q: %w", addr, err)
	}
	if host != "127.0.0.1" {
		return nil, fmt.Errorf("dashboard must bind to 127.0.0.1, got %q", host)
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}
	return listener, nil
}

func writeSession(path string, session Session) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dashboard session dir: %w", err)
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal dashboard session: %w", err)
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write dashboard session: %w", err)
	}
	return nil
}
