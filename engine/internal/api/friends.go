package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/metadata"
	"github.com/TheGuyDangerous/Syncy/engine/internal/syncengine"
)

const friendNetworkTimeout = 30 * time.Second

func (s *Server) handleInvite(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"code": s.engine.InviteCode()})
}

func (s *Server) handleAddFriend(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(body.Code) == "" {
		writeError(w, http.StatusBadRequest, "invite code is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), friendNetworkTimeout)
	defer cancel()
	dev, delivered, err := s.engine.AddFriendByCode(ctx, body.Code)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		Device    core.Device `json:"device"`
		Delivered bool        `json:"delivered"`
	}{dev, delivered})
}

func (s *Server) handleFriendFolders(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), friendNetworkTimeout)
	defer cancel()
	folders, err := s.engine.FriendFolders(ctx, core.DeviceID(r.PathValue("id")))
	switch {
	case errors.Is(err, metadata.ErrNotFound):
		writeError(w, http.StatusNotFound, "no such device")
	case errors.Is(err, syncengine.ErrNotFriend):
		writeError(w, http.StatusBadRequest, "that device is not a friend yet")
	case errors.Is(err, syncengine.ErrUnreachable):
		writeError(w, http.StatusBadGateway, "friend is offline or unreachable right now")
	case err != nil:
		writeError(w, http.StatusInternalServerError, err.Error())
	default:
		writeJSON(w, http.StatusOK, folders)
	}
}

func (s *Server) handleListFriendRequests(w http.ResponseWriter, _ *http.Request) {
	reqs, err := s.engine.FriendRequests()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if reqs == nil {
		reqs = []core.FriendRequest{}
	}
	writeJSON(w, http.StatusOK, reqs)
}

func (s *Server) handleAcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), friendNetworkTimeout)
	defer cancel()
	dev, notified, err := s.engine.AcceptFriendRequest(ctx, core.DeviceID(r.PathValue("id")))
	if errors.Is(err, metadata.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no such friend request")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Device   core.Device `json:"device"`
		Notified bool        `json:"notified"`
	}{dev, notified})
}

func (s *Server) handleRejectFriendRequest(w http.ResponseWriter, r *http.Request) {
	err := s.engine.RejectFriendRequest(core.DeviceID(r.PathValue("id")))
	if errors.Is(err, metadata.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no such friend request")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetDiscovery(w http.ResponseWriter, _ *http.Request) {
	settings, err := s.engine.DiscoverySettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handlePutDiscovery(w http.ResponseWriter, r *http.Request) {
	var settings core.DiscoverySettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.engine.SetDiscoverySettings(settings); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}
