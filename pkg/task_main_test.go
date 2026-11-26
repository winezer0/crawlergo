package pkg

import (
	"testing"

	"github.com/Qianlitp/crawlergo/pkg/model"
)

func TestUniqueOriginTargetsDeduplicatesBySchemeAndHost(t *testing.T) {
	targets := []*model.Request{
		originRequest(t, "https://shop.example.com/one"),
		originRequest(t, "https://shop.example.com/two"),
		originRequest(t, "http://shop.example.com/"),
		originRequest(t, "https://api.example.com/"),
	}
	got := uniqueOriginTargets(targets)
	if len(got) != 3 {
		t.Fatalf("uniqueOriginTargets() returned %d origins, want 3", len(got))
	}
	if got[0] != targets[0] || got[1] != targets[2] || got[2] != targets[3] {
		t.Fatalf("uniqueOriginTargets() = %#v, want first request for each origin", got)
	}
}

func originRequest(t *testing.T, rawURL string) *model.Request {
	t.Helper()
	parsed, err := model.GetUrl(rawURL)
	if err != nil {
		t.Fatalf("GetUrl(%q) error = %v", rawURL, err)
	}
	request := model.GetRequest("GET", parsed)
	return &request
}
