package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/ai"
)

type aiConfigView struct {
	Enabled bool    `json:"enabled"`
	Kind    ai.Kind `json:"kind"`
	BaseURL string  `json:"base_url"`
	Model   string  `json:"model"`
	HasKey  bool    `json:"has_api_key"`
}

func loadAIConfig(path string) (ai.Config, error) {
	if path == "" {
		return ai.Config{}, errors.New("api: no ai config path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ai.Config{}, err
	}
	var cfg ai.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ai.Config{}, err
	}
	return cfg, nil
}

func saveAIConfig(path string, cfg ai.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *Server) aiConfig() ai.Config {
	s.aiMu.Lock()
	defer s.aiMu.Unlock()
	return s.aiCfg
}

func viewOf(cfg ai.Config) aiConfigView {
	return aiConfigView{
		Enabled: cfg.Enabled,
		Kind:    cfg.Kind,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		HasKey:  cfg.APIKey != "",
	}
}

func (s *Server) handleGetAI(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, viewOf(s.aiConfig()))
}

func (s *Server) handleSaveAI(w http.ResponseWriter, r *http.Request) {
	var incoming ai.Config
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s.aiMu.Lock()
	if incoming.APIKey == "" {
		incoming.APIKey = s.aiCfg.APIKey
	}
	s.aiMu.Unlock()

	if incoming.Enabled {
		if err := incoming.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if err := saveAIConfig(s.aiPath, incoming); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.aiMu.Lock()
	s.aiCfg = incoming
	s.aiMu.Unlock()
	writeJSON(w, http.StatusOK, viewOf(incoming))
}

func (s *Server) handleTestAI(w http.ResponseWriter, r *http.Request) {
	var cfg ai.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s.aiMu.Lock()
	if cfg.APIKey == "" {
		cfg.APIKey = s.aiCfg.APIKey
	}
	s.aiMu.Unlock()
	cfg.Enabled = true
	if err := cfg.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := ai.New(cfg).TestConnection(ctx); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleExplainConflict(w http.ResponseWriter, r *http.Request) {
	var d ai.ConflictDetails
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s.aiComplete(w, r, func(ctx context.Context, c *ai.Client) (string, error) {
		return c.ExplainConflict(ctx, d)
	})
}

func (s *Server) handleAnalyzeLogs(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Logs string `json:"logs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s.aiComplete(w, r, func(ctx context.Context, c *ai.Client) (string, error) {
		return c.AnalyzeLogs(ctx, body.Logs)
	})
}

func (s *Server) aiComplete(w http.ResponseWriter, r *http.Request, fn func(context.Context, *ai.Client) (string, error)) {
	cfg := s.aiConfig()
	if !cfg.Enabled {
		writeError(w, http.StatusPreconditionFailed, ai.ErrDisabled.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	text, err := fn(ctx, ai.New(cfg))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}
