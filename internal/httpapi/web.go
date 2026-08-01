package httpapi

import (
	"bytes"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"sort"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

//go:embed webdist/*
var webdist embed.FS

// WebHandler serves the Vite build embedded in the binary.
func WebHandler() http.Handler {
	assets, err := fs.Sub(webdist, "webdist")
	if err != nil {
		panic(err)
	}
	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		panic(err)
	}
	return &embeddedWebHandler{files: http.FileServer(http.FS(assets)), index: index}
}

type embeddedWebHandler struct {
	files http.Handler
	index []byte
}

func (h *embeddedWebHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet && request.URL.Path == "/" {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(h.index)
		return
	}
	h.files.ServeHTTP(writer, request)
}

type webRuntimeConfig struct {
	CSRFToken    string                 `json:"csrfToken"`
	Capabilities webRuntimeCapabilities `json:"capabilities"`
}

type webRuntimeCapabilities struct {
	CreateRuns       bool           `json:"createRuns"`
	CancelRuns       bool           `json:"cancelRuns"`
	Approvals        bool           `json:"approvals"`
	Artifacts        bool           `json:"artifacts"`
	ConfigValidation bool           `json:"configValidation"`
	Credentials      bool           `json:"credentials"`
	Demo             bool           `json:"demo"`
	FixedRuns        []domain.RunID `json:"fixedRuns"`
}

func localRuntimeWebHandler(next http.Handler, csrfToken string, capabilities Capabilities) http.Handler {
	embedded, ok := next.(*embeddedWebHandler)
	if !ok {
		return next
	}
	fixedRuns := make([]domain.RunID, 0, len(capabilities.FixedRuns))
	for runID := range capabilities.FixedRuns {
		fixedRuns = append(fixedRuns, runID)
	}
	sort.Slice(fixedRuns, func(left, right int) bool { return fixedRuns[left] < fixedRuns[right] })
	config, err := json.Marshal(webRuntimeConfig{
		CSRFToken: csrfToken,
		Capabilities: webRuntimeCapabilities{
			CreateRuns: capabilities.CreateRuns, CancelRuns: capabilities.CancelRuns,
			Approvals: capabilities.Approvals, Artifacts: capabilities.Artifacts,
			ConfigValidation: capabilities.ConfigValidation, Credentials: capabilities.Credentials,
			Demo: false, FixedRuns: fixedRuns,
		},
	})
	if err != nil {
		panic(err)
	}
	script := []byte("<script>window.__AI4SE_RUNTIME__=" + string(config) + ";</script>")
	index := bytes.LastIndex(embedded.index, []byte("</head>"))
	if index < 0 {
		return next
	}
	withConfig := make([]byte, 0, len(embedded.index)+len(script))
	withConfig = append(withConfig, embedded.index[:index]...)
	withConfig = append(withConfig, script...)
	withConfig = append(withConfig, embedded.index[index:]...)
	configured := *embedded
	configured.index = withConfig
	return &configured
}
