// Package invite encodes a device's identity and reachable endpoints as a
// compact copy-pasteable code that another device can use to send a friend
// request.
package invite

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
)

const Prefix = "SYNCY1-"

var (
	ErrNotInvite = errors.New("invite: not a Syncy invite code")
	ErrMissingID = errors.New("invite: code has no device id")
)

type Code struct {
	DeviceID  string   `json:"id"`
	Name      string   `json:"name,omitempty"`
	Endpoints []string `json:"eps,omitempty"`
}

func Encode(c Code) string {
	payload, _ := json.Marshal(c)
	return Prefix + base64.RawURLEncoding.EncodeToString(payload)
}

func Decode(token string) (Code, error) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, Prefix) {
		return Code{}, ErrNotInvite
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, Prefix))
	if err != nil {
		return Code{}, fmt.Errorf("invite: undecodable code: %w", err)
	}
	var c Code
	if err := json.Unmarshal(payload, &c); err != nil {
		return Code{}, fmt.Errorf("invite: malformed code: %w", err)
	}
	if strings.TrimSpace(c.DeviceID) == "" {
		return Code{}, ErrMissingID
	}
	for _, ep := range c.Endpoints {
		if _, _, err := net.SplitHostPort(ep); err != nil {
			return Code{}, fmt.Errorf("invite: bad endpoint %q", ep)
		}
	}
	return c, nil
}
