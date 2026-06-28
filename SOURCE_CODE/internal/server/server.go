package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/nossh/nossh/internal/config"
	"github.com/nossh/nossh/internal/protocol"
	"github.com/nossh/nossh/internal/registry"
)

type Server struct {
	cfg      config.Server
	reg      *registry.Registry
	sessions sync.Map // uuid -> *agentSession
}

type agentSession struct {
	uuid     string
	code     string
	hostname string
	name     string
	status   protocol.AgentStatus
	session  *yamux.Session
	lastPing time.Time
	mu       sync.Mutex
}

func New(cfg config.Server, reg *registry.Registry) *Server {
	return &Server{cfg: cfg, reg: reg}
}

func (s *Server) Run(ctx context.Context) error {
	agentLn, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.cfg.ListenAddr, s.cfg.AgentPort))
	if err != nil {
		return fmt.Errorf("listen agent port: %w", err)
	}
	clientLn, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.cfg.ListenAddr, s.cfg.ClientPort))
	if err != nil {
		_ = agentLn.Close()
		return fmt.Errorf("listen client port: %w", err)
	}

	adminSrv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.cfg.AdminPort),
		Handler: s.adminHandler(),
	}

	errCh := make(chan error, 3)
	go func() { errCh <- s.serveAgents(ctx, agentLn) }()
	go func() { errCh <- s.serveClients(ctx, clientLn) }()
	go func() { errCh <- adminSrv.ListenAndServe() }()

	go func() {
		<-ctx.Done()
		_ = agentLn.Close()
		_ = clientLn.Close()
		_ = adminSrv.Shutdown(context.Background())
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) serveAgents(ctx context.Context, ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		go s.handleAgent(ctx, conn)
	}
}

func (s *Server) handleAgent(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	session, err := yamux.Server(conn, nil)
	if err != nil {
		log.Printf("agent yamux: %v", err)
		return
	}
	defer session.Close()

	control, err := session.Accept()
	if err != nil {
		log.Printf("agent control stream: %v", err)
		return
	}

	var req protocol.RegisterRequest
	if err := protocol.ReadJSON(control, &req); err != nil || req.Type != "register" {
		log.Printf("invalid register: %v", err)
		return
	}
	if req.Token != s.cfg.InstallToken {
		_ = protocol.WriteJSON(control, protocol.RegisterResponse{
			Type:    "register",
			Status:  protocol.StatusRevoked,
			Message: "invalid install token",
		})
		return
	}

	rec, err := s.reg.UpsertAgent(req.UUID, req.Code, req.Hostname)
	if err != nil {
		log.Printf("registry upsert: %v", err)
		return
	}

	as := &agentSession{
		uuid:     req.UUID,
		code:     req.Code,
		hostname: req.Hostname,
		name:     rec.Name,
		status:   rec.Status,
		session:  session,
		lastPing: time.Now(),
	}
	s.sessions.Store(req.UUID, as)
	defer s.sessions.Delete(req.UUID)

	resp := protocol.RegisterResponse{
		Type:   "register",
		Status: rec.Status,
		Name:   rec.Name,
	}
	if rec.Status == protocol.StatusActive {
		resp.Message = "connected"
	} else {
		resp.Message = "pending admin approval"
	}
	if err := protocol.WriteJSON(control, resp); err != nil {
		return
	}

	go s.acceptAgentStreams(session)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := protocol.WriteJSON(control, protocol.NewPing()); err != nil {
				return
			}
			as.mu.Lock()
			as.lastPing = time.Now()
			as.mu.Unlock()
			_ = s.reg.Touch(req.UUID)
		case <-session.CloseChan():
			return
		}
	}
}

func (s *Server) acceptAgentStreams(session *yamux.Session) {
	for {
		stream, err := session.Accept()
		if err != nil {
			return
		}
		// Agent-initiated streams are not used in this version.
		stream.Close()
	}
}

func (s *Server) serveClients(ctx context.Context, ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	line = strings.TrimSpace(line)
	parts := strings.Fields(line)
	if len(parts) != 2 {
		_, _ = io.WriteString(conn, "ERR invalid handshake\n")
		return
	}

	cmd := strings.ToUpper(parts[0])
	name := parts[1]

	switch cmd {
	case "CHECK":
		s.handleClientCheck(conn, name)
	case "CONNECT":
		s.handleClientConnect(conn, reader, name)
	default:
		_, _ = io.WriteString(conn, "ERR invalid handshake\n")
	}
}

func (s *Server) handleClientCheck(conn net.Conn, name string) {
	rec, err := s.reg.GetActiveByNameOrCode(name)
	if err != nil {
		if err.Error() == "agent not active" {
			_, _ = io.WriteString(conn, "ERR not active\n")
			return
		}
		_, _ = io.WriteString(conn, "ERR not found\n")
		return
	}
	_ = rec
	_, _ = io.WriteString(conn, "OK\n")
}

func (s *Server) handleClientConnect(conn net.Conn, reader *bufio.Reader, name string) {
	rec, err := s.reg.GetActiveByNameOrCode(name)
	if err != nil {
		_, _ = io.WriteString(conn, "ERR machine not available\n")
		return
	}

	raw, ok := s.sessions.Load(rec.UUID)
	if !ok {
		_, _ = io.WriteString(conn, "ERR machine offline\n")
		return
	}
	as := raw.(*agentSession)

	stream, err := as.session.Open()
	if err != nil {
		_, _ = io.WriteString(conn, "ERR relay failed\n")
		return
	}
	defer stream.Close()

	if _, err := io.WriteString(stream, "BRIDGE\n"); err != nil {
		_, _ = io.WriteString(conn, "ERR relay failed\n")
		return
	}

	_, _ = io.WriteString(conn, "OK\n")
	_ = conn.SetDeadline(time.Time{})

	go func() { _, _ = io.Copy(stream, reader); stream.Close() }()
	_, _ = io.Copy(conn, stream)
}

func (s *Server) listAgents() []registry.AgentRecord {
	records, err := s.reg.List()
	if err != nil {
		return nil
	}
	online := map[string]bool{}
	s.sessions.Range(func(key, value any) bool {
		online[key.(string)] = true
		return true
	})
	for i := range records {
		records[i].Online = online[records[i].UUID]
	}
	return records
}

func (s *Server) getSessionByCode(code string) (*agentSession, bool) {
	var found *agentSession
	s.sessions.Range(func(_, value any) bool {
		as := value.(*agentSession)
		if as.code == code {
			found = as
			return false
		}
		return true
	})
	return found, found != nil
}
