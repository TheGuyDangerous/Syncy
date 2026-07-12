package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ConflictDetails struct {
	Folder         string    `json:"folder"`
	Path           string    `json:"path"`
	LocalDevice    string    `json:"local_device"`
	RemoteDevice   string    `json:"remote_device"`
	LocalModified  time.Time `json:"local_modified"`
	RemoteModified time.Time `json:"remote_modified"`
	LocalSize      int64     `json:"local_size"`
	RemoteSize     int64     `json:"remote_size"`
}

const assistantRole = "You are a concise assistant built into Syncy, a local-first peer-to-peer file sync tool. Answer plainly for a non-expert user. Never invent details beyond what you are given."

func (c *Client) ExplainConflict(ctx context.Context, d ConflictDetails) (string, error) {
	system := assistantRole + " Explain why this file is in conflict and recommend which copy to keep. Keep it under 120 words."
	return c.Complete(ctx, system, conflictPrompt(d))
}

func (c *Client) AnalyzeLogs(ctx context.Context, logs string) (string, error) {
	system := assistantRole + " Summarize these engine logs: call out errors, likely causes, and next steps. Keep it under 150 words."
	if len(logs) > 12000 {
		logs = logs[len(logs)-12000:]
	}
	return c.Complete(ctx, system, "Logs:\n"+logs)
}

func (c *Client) TestConnection(ctx context.Context) error {
	reply, err := c.Complete(ctx, "You are a connection check.", "Reply with the single word: ok")
	if err != nil {
		return err
	}
	if strings.TrimSpace(reply) == "" {
		return ErrEmptyReply
	}
	return nil
}

func conflictPrompt(d ConflictDetails) string {
	var b strings.Builder
	fmt.Fprintf(&b, "A sync conflict happened in folder %q for file %q.\n", d.Folder, d.Path)
	fmt.Fprintf(&b, "Local copy: device %s, modified %s, %d bytes.\n", short(d.LocalDevice), stamp(d.LocalModified), d.LocalSize)
	fmt.Fprintf(&b, "Remote copy: device %s, modified %s, %d bytes.\n", short(d.RemoteDevice), stamp(d.RemoteModified), d.RemoteSize)
	b.WriteString("Both versions changed independently since they last agreed.")
	return b.String()
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	if id == "" {
		return "unknown"
	}
	return id
}

func stamp(t time.Time) string {
	if t.IsZero() {
		return "unknown time"
	}
	return t.UTC().Format(time.RFC3339)
}
