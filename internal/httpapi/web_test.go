package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/httpapi"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
)

func TestEmbeddedWebAssetsAreReachableThroughDemoRouter(t *testing.T) {
	router, err := httpapi.NewDemo(httpapi.Options{
		Application: webApplication{}, Store: store.NewMemory(),
		Capabilities: httpapi.DemoCapabilities(), AppShell: httpapi.WebHandler(),
	})
	if err != nil {
		t.Fatal(err)
	}
	root := httptest.NewRecorder()
	router.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Code != http.StatusOK {
		t.Fatalf("root = %d: %s", root.Code, root.Body.String())
	}
	match := regexp.MustCompile(`(?:src|href)="(/assets/[^"]+)"`).FindStringSubmatch(root.Body.String())
	if len(match) != 2 {
		t.Fatalf("asset URL missing from %s", root.Body.String())
	}
	asset := httptest.NewRecorder()
	router.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, match[1], nil))
	if asset.Code != http.StatusOK || asset.Body.Len() == 0 {
		t.Fatalf("asset %s = %d (%d bytes)", match[1], asset.Code, asset.Body.Len())
	}
}

type webApplication struct{}

func (webApplication) CreateRun(context.Context, app.CreateRunRequest) (domain.Run, error) {
	return domain.Run{}, nil
}
func (webApplication) GetRun(context.Context, domain.RunID) (domain.Run, error) {
	return domain.Run{}, nil
}
func (webApplication) CancelRun(context.Context, domain.RunID) error            { return nil }
func (webApplication) Approve(context.Context, domain.RunID, string) error      { return nil }
func (webApplication) Reject(context.Context, domain.RunID, string, bool) error { return nil }
func (webApplication) Preflight(context.Context, app.CreateRunRequest) app.PreflightReport {
	return app.PreflightReport{}
}
