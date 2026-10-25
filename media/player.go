package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/example/velvetcut/editor"
)

// EDL serializes the non-destructive timeline as an mpv virtual movie.
// The byte-length prefix is essential for names containing commas and semicolons.
func EDL(p editor.Project) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	if len(p.Clips) == 0 {
		return "", errors.New("empty movie")
	}
	var out strings.Builder
	out.WriteString("# mpv EDL v0\n")
	for _, c := range p.Clips {
		name, err := filepath.Abs(c.Path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&out, "%%%d%%%s,%s,%s\n", len(name), name, strconv.FormatFloat(c.SourceStart, 'f', 6, 64), strconv.FormatFloat(c.Duration(), 'f', 6, 64))
	}
	return out.String(), nil
}

// Player runs mpv in a separate video window, using mpv's own decoder and audio
// clock. The Fyne canvas is used for scrubbing thumbnails, not live video.
// All IPC is over an ephemeral local Unix socket in a private directory.
type Player struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	temp   string
	socket string
	closed bool
}

func StartPlayer(ctx context.Context, project editor.Project, at float64) (*Player, error) {
	if _, err := exec.LookPath("mpv"); err != nil {
		return nil, fmt.Errorf("install mpv for synchronized video/audio preview: %w", err)
	}
	content, err := EDL(project)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "velvetcut-preview-")
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	edl := filepath.Join(dir, "timeline.edl")
	if err := os.WriteFile(edl, []byte(content), 0600); err != nil {
		cleanup()
		return nil, err
	}
	sock := filepath.Join(dir, "ipc.sock")
	args := []string{"--no-config", "--force-window=yes", "--idle=no", "--keep-open=yes", "--osc=no", "--input-terminal=no", "--title=VelvetCut Preview", "--input-ipc-server=" + sock, "--start=" + strconv.FormatFloat(at, 'f', 5, 64), edl}
	cmd := exec.CommandContext(ctx, "mpv", args...)
	if err := cmd.Start(); err != nil {
		cleanup()
		return nil, err
	}
	player := &Player{cmd: cmd, temp: dir, socket: sock}
	// Reap the process even if it exits without an explicit Close.
	go func() { _ = cmd.Wait(); player.mu.Lock(); player.closed = true; player.mu.Unlock() }()
	readyCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	for {
		if readyCtx.Err() != nil {
			player.Close()
			return nil, fmt.Errorf("mpv did not start its IPC socket: %w", readyCtx.Err())
		}
		player.mu.Lock()
		closed := player.closed
		player.mu.Unlock()
		if closed {
			player.Close()
			return nil, errors.New("mpv exited before preview started")
		}
		if _, err := os.Stat(sock); err == nil {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	return player, nil
}

func (p *Player) command(ctx context.Context, args ...any) (json.RawMessage, error) {
	p.mu.Lock()
	closed := p.closed
	socket := p.socket
	p.mu.Unlock()
	if closed {
		return nil, errors.New("mpv has exited")
	}
	d := net.Dialer{Timeout: 400 * time.Millisecond}
	conn, err := d.DialContext(ctx, "unix", socket)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(700 * time.Millisecond))
	if err := json.NewEncoder(conn).Encode(map[string]any{"command": args, "request_id": 7}); err != nil {
		return nil, err
	}
	var resp struct {
		RequestID int             `json:"request_id"`
		Error     string          `json:"error"`
		Data      json.RawMessage `json:"data"`
	}
	dec := json.NewDecoder(conn)
	for {
		if err := dec.Decode(&resp); err != nil {
			return nil, err
		}
		if resp.RequestID != 7 {
			continue
		} // async mpv events do not match a request
		if resp.Error != "success" {
			return nil, fmt.Errorf("mpv: %s", resp.Error)
		}
		return resp.Data, nil
	}
}
func (p *Player) Position(ctx context.Context) (float64, error) {
	raw, err := p.command(ctx, "get_property", "time-pos")
	if err != nil {
		return 0, err
	}
	var pos float64
	if err := json.Unmarshal(raw, &pos); err != nil {
		return 0, err
	}
	return pos, nil
}
func (p *Player) Pause(ctx context.Context, pause bool) error {
	_, err := p.command(ctx, "set_property", "pause", pause)
	return err
}
func (p *Player) Seek(ctx context.Context, seconds float64) error {
	_, err := p.command(ctx, "seek", seconds, "absolute+exact")
	return err
}
func (p *Player) Close() {
	p.mu.Lock()
	if !p.closed && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	p.closed = true
	p.mu.Unlock()
	// Process wait happens in StartPlayer's goroutine.
	_ = os.RemoveAll(p.temp)
}
