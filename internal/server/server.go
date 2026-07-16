package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/unlatch-ai/rolloutviz/internal/model"
	webassets "github.com/unlatch-ai/rolloutviz/web"
)

const fallbackUI = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>RolloutViz</title><style>body{font:16px system-ui;margin:3rem;max-width:70ch}code{background:#eee;padding:.15rem .3rem}</style></head>
<body><h1>RolloutViz</h1><p>The local viewer is running.</p><p>Trajectory data is available at <code>/api/v1/trajectory</code>.</p></body></html>`

type Document struct {
	Trajectory *model.Trajectory `json:"trajectory"`
	Events     []*model.Event    `json:"events"`
	Run        *model.Run        `json:"run,omitempty"`
	Case       *model.Case       `json:"case,omitempty"`
	Group      *model.Group      `json:"group,omitempty"`
	Signals    []*model.Signal   `json:"signals,omitempty"`
	Artifacts  []*model.Artifact `json:"artifacts,omitempty"`
}

// LoadCanonicalNDJSON reads canonical records without changing the source.
func LoadCanonicalNDJSON(path string) (Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return Document{}, fmt.Errorf("open canonical trajectory: %w", err)
	}
	defer file.Close()

	document := Document{
		Events: make([]*model.Event, 0), Signals: make([]*model.Signal, 0),
		Artifacts: make([]*model.Artifact, 0),
	}
	allEvents := make([]*model.Event, 0)
	allSignals := make([]*model.Signal, 0)
	allArtifacts := make([]*model.Artifact, 0)
	runs := make(map[string]*model.Run)
	cases := make(map[string]*model.Case)
	groups := make(map[string]*model.Group)
	if err := model.Decode(file, func(record *model.Record) error {
		switch value := record.Value.(type) {
		case *model.Run:
			runs[value.ID] = value
		case *model.Case:
			cases[value.ID] = value
		case *model.Group:
			groups[value.ID] = value
		case *model.Trajectory:
			if document.Trajectory == nil {
				document.Trajectory = value
			}
		case *model.Event:
			// Keep the canonical source line available to the inspector even
			// when the adapter did not provide a separate raw payload.
			if len(value.Raw) == 0 {
				value.Raw = append(json.RawMessage(nil), record.Raw...)
			}
			allEvents = append(allEvents, value)
		case *model.Signal:
			allSignals = append(allSignals, value)
		case *model.Artifact:
			allArtifacts = append(allArtifacts, value)
		}
		return nil
	}); err != nil {
		return Document{}, fmt.Errorf("decode canonical trajectory: %w", err)
	}
	if document.Trajectory == nil {
		return Document{}, fmt.Errorf("canonical trajectory contains no trajectory record")
	}
	for _, event := range allEvents {
		if event.TrajectoryID == document.Trajectory.ID {
			document.Events = append(document.Events, event)
		}
	}
	for _, signal := range allSignals {
		if signal.TrajectoryID == document.Trajectory.ID {
			document.Signals = append(document.Signals, signal)
		}
	}
	for _, artifact := range allArtifacts {
		if artifact.TrajectoryID == document.Trajectory.ID {
			document.Artifacts = append(document.Artifacts, artifact)
		}
	}
	document.Group = groups[document.Trajectory.GroupID]
	if document.Group != nil {
		document.Case = cases[document.Group.CaseID]
	}
	if document.Case != nil {
		document.Run = runs[document.Case.RunID]
	}
	return document, nil
}

func ListenLoopback(port int) (net.Listener, error) {
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("port must be between 0 and 65535")
	}
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen on loopback: %w", err)
	}
	return listener, nil
}

func NewHandler(document Document) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/trajectory", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(response).Encode(document); err != nil {
			http.Error(response, "encode trajectory response", http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("GET /api/v1/health", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"status":"ok"}`+"\n")
	})
	mux.Handle("GET /", viewerHandler())
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		mux.ServeHTTP(response, request)
	})
}

func viewerHandler() http.Handler {
	dist, err := fs.Sub(webassets.Dist, "dist")
	if err != nil {
		return fallbackHandler()
	}
	if _, err := fs.Stat(dist, "index.html"); err != nil {
		return fallbackHandler()
	}

	files := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			http.NotFound(response, request)
			return
		}
		name := strings.TrimPrefix(request.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if info, err := fs.Stat(dist, name); err == nil && !info.IsDir() {
			files.ServeHTTP(response, request)
			return
		}

		// Viewer URLs are client-side deep links. Unknown asset-like paths stay
		// 404s, while route paths receive the application shell.
		if strings.Contains(name, ".") {
			http.NotFound(response, request)
			return
		}
		request.URL.Path = "/"
		files.ServeHTTP(response, request)
	})
}

func fallbackHandler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/" {
			http.NotFound(response, request)
			return
		}
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(response, fallbackUI)
	})
}
