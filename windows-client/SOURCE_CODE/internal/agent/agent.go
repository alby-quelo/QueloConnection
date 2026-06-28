package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/nossh/nossh/internal/config"
	"github.com/nossh/nossh/internal/protocol"
)

const reconnectBase = 5 * time.Second
const reconnectMax = 60 * time.Second

type Runner struct {
	cfg config.Agent
}

func New(cfg config.Agent) *Runner {
	return &Runner{cfg: cfg}
}

func (r *Runner) Run(ctx context.Context) error {
	delay := reconnectBase
	for {
		err := r.connectOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Printf("disconnected: %v; reconnect in %s", err, delay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > reconnectMax {
			delay = reconnectMax
		}
	}
}

func (r *Runner) connectOnce(ctx context.Context) error {
	conn, err := net.DialTimeout("tcp", r.cfg.ServerURL, 15*time.Second)
	if err != nil {
		return fmt.Errorf("dial server: %w", err)
	}
	defer conn.Close()

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return fmt.Errorf("yamux client: %w", err)
	}
	defer session.Close()

	control, err := session.Open()
	if err != nil {
		return fmt.Errorf("open control stream: %w", err)
	}

	req := protocol.NewRegisterRequest(r.cfg.Code, r.cfg.UUID, hostname(), r.cfg.InstallToken)
	if err := protocol.WriteJSON(control, req); err != nil {
		return fmt.Errorf("register: %w", err)
	}

	var resp protocol.RegisterResponse
	if err := protocol.ReadJSON(control, &resp); err != nil {
		return fmt.Errorf("register response: %w", err)
	}
	if resp.Message == "invalid install token" {
		return fmt.Errorf("invalid install token")
	}
	log.Printf("registered: status=%s name=%q", resp.Status, resp.Name)

	errCh := make(chan error, 2)
	go func() { errCh <- r.serveStreams(session) }()
	go func() { errCh <- r.readControl(ctx, control) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	case <-session.CloseChan():
		return fmt.Errorf("session closed")
	}
}

func (r *Runner) readControl(ctx context.Context, control net.Conn) error {
	dec := bufio.NewReader(control)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var msg protocol.PingMessage
		if err := protocol.ReadJSON(dec, &msg); err != nil {
			return err
		}
	}
}

func (r *Runner) serveStreams(session *yamux.Session) error {
	for {
		stream, err := session.Accept()
		if err != nil {
			return err
		}
		go r.bridgeSSH(stream)
	}
}

func (r *Runner) bridgeSSH(stream net.Conn) {
	defer stream.Close()
	reader := bufio.NewReader(stream)
	line, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "BRIDGE" {
		return
	}

	target := fmt.Sprintf("127.0.0.1:%d", r.cfg.SSHPort)
	sshConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("bridge to ssh: %v", err)
		return
	}
	defer sshConn.Close()

	go func() { _, _ = io.Copy(sshConn, reader); sshConn.Close() }()
	_, _ = io.Copy(stream, sshConn)
}

func hostname() string {
	name, err := osHostLookup()
	if err != nil {
		return "unknown"
	}
	return name
}
