// Package api serves the daemon's local control API over loopback, authenticated
// with a bearer token, so desktop and mobile clients can drive the engine.
package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/syncengine"
)

type Server struct {
	engine *syncengine.Engine
	token  string
	mux    *http.ServeMux
}

func New(engine *syncengine.Engine, token string) *Server {
	s := &Server{engine: engine, token: token, mux: http.NewServeMux()}
	s.routes()
	return s
}

func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /status", s.handleStatus)
	s.mux.HandleFunc("GET /folders", s.handleListFolders)
	s.mux.HandleFunc("POST /folders", s.handleAddFolder)
	s.mux.HandleFunc("DELETE /folders/{id}", s.handleRemoveFolder)
	s.mux.HandleFunc("GET /folders/{id}/versions", s.handleVersions)
	s.mux.HandleFunc("GET /devices", s.handleListDevices)
	s.mux.HandleFunc("GET /conflicts", s.handleConflicts)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) authorized(r *http.Request) bool {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || h[:len(prefix)] != prefix {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(h[len(prefix):]), []byte(s.token)) == 1
}

type statusResponse struct {
	DeviceID string `json:"device_id"`
	Folders  int    `json:"folders"`
	Devices  int    `json:"devices"`
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	folders, err := s.engine.Folders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	devices, err := s.engine.Devices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{
		DeviceID: string(s.engine.ID()),
		Folders:  len(folders),
		Devices:  len(devices),
	})
}

func (s *Server) handleListFolders(w http.ResponseWriter, _ *http.Request) {
	folders, err := s.engine.Folders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, folders)
}

func (s *Server) handleAddFolder(w http.ResponseWriter, r *http.Request) {
	var f core.Folder
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.engine.AddFolder(f); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (s *Server) handleRemoveFolder(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.RemoveFolder(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleVersions(w http.ResponseWriter, r *http.Request) {
	versions, err := s.engine.FolderVersions(r.PathValue("id"), r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, versions)
}

func (s *Server) handleListDevices(w http.ResponseWriter, _ *http.Request) {
	devices, err := s.engine.Devices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (s *Server) handleConflicts(w http.ResponseWriter, _ *http.Request) {
	conflicts, err := s.engine.Conflicts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conflicts)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
