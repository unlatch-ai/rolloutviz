package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/unlatch-ai/rolloutviz/internal/server"
)

type Viewer struct {
	SourcePath string
	Port       int
}

type RunningViewer struct {
	URL        string
	SourcePath string
	Listener   net.Listener
	Server     *http.Server
}

// StartViewer validates and parses a source before opening a loopback listener.
// The caller owns the returned server lifecycle.
func StartViewer(config Viewer) (*RunningViewer, error) {
	path, err := ValidateSource(config.SourcePath)
	if err != nil {
		return nil, err
	}
	document, err := server.LoadCanonicalNDJSON(path)
	if err != nil {
		return nil, err
	}
	listener, err := server.ListenLoopback(config.Port)
	if err != nil {
		return nil, err
	}
	httpServer := &http.Server{
		Handler:           server.NewHandler(document),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return &RunningViewer{
		URL:        fmt.Sprintf("http://%s", listener.Addr().String()),
		SourcePath: path,
		Listener:   listener,
		Server:     httpServer,
	}, nil
}

func (viewer *RunningViewer) Serve() error {
	err := viewer.Server.Serve(viewer.Listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (viewer *RunningViewer) Shutdown(ctx context.Context) error {
	return viewer.Server.Shutdown(ctx)
}
