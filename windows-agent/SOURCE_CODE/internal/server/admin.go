package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nossh/nossh/internal/codes"
	"github.com/nossh/nossh/internal/protocol"
)

func (s *Server) adminHandler() http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("/api/agents", s.handleAgents)
	api.HandleFunc("/api/agents/", s.handleAgentByCode)
	api.HandleFunc("/api/rename", s.handleRename)
	api.HandleFunc("/api/revoke", s.handleRevoke)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/terminal/ws" {
			s.handleTerminalWS(w, r)
			return
		}
		if !s.authorize(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		api.ServeHTTP(w, r)
	})
}

func (s *Server) authorize(r *http.Request) bool {
	if s.cfg.AdminToken == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ") == s.cfg.AdminToken
	}
	return r.Header.Get("X-Admin-Token") == s.cfg.AdminToken
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.listAgents())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAgentByCode(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	code := codes.Normalize(parts[0])

	switch {
	case len(parts) == 2 && parts[1] == "name" && r.Method == http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		rec, err := s.reg.AssignName(code, strings.TrimSpace(body.Name))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.refreshSessionName(rec.UUID, rec.Name, rec.Status)
		writeJSON(w, rec)

	case len(parts) == 1 && r.Method == http.MethodDelete:
		if err := s.reg.Delete(code); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.NotFound(w, r)
	}
}

type renameRequest struct {
	Name    string `json:"name"`
	NewName string `json:"new_name"`
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body renameRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rec, err := s.reg.Rename(body.Name, body.NewName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.refreshSessionName(rec.UUID, rec.Name, rec.Status)
	writeJSON(w, rec)
}

func (s *Server) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := s.reg.Revoke(body.Name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) refreshSessionName(uuid, name string, status protocol.AgentStatus) {
	raw, ok := s.sessions.Load(uuid)
	if !ok {
		return
	}
	as := raw.(*agentSession)
	as.mu.Lock()
	as.name = name
	as.status = status
	as.mu.Unlock()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
