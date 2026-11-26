package pkg

import (
	"reflect"
	"testing"

	"github.com/Qianlitp/crawlergo/pkg/model"
)

func TestRootDomainsAndSubdomainsCollectAcrossTargets(t *testing.T) {
	targets := []*model.Request{
		domainRequest(t, "https://app.example.com/"),
		domainRequest(t, "https://api.example.co.uk/"),
		domainRequest(t, "https://other.example.com/"),
	}
	roots := RootDomainsForTargets(targets)
	if !reflect.DeepEqual(roots, []string{"example.com", "example.co.uk"}) {
		t.Fatalf("RootDomainsForTargets() = %#v", roots)
	}

	requests := []*model.Request{
		domainRequest(t, "https://example.com/"),
		domainRequest(t, "https://cdn.example.com/"),
		domainRequest(t, "https://example.co.uk/"),
		domainRequest(t, "https://assets.example.co.uk/"),
		domainRequest(t, "https://example.org/"),
	}
	got := SubDomainCollectForRoots(requests, roots)
	want := []string{"cdn.example.com", "assets.example.co.uk"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SubDomainCollectForRoots() = %#v, want %#v", got, want)
	}
}

func domainRequest(t *testing.T, rawURL string) *model.Request {
	t.Helper()
	parsed, err := model.GetUrl(rawURL)
	if err != nil {
		t.Fatalf("GetUrl(%q) error = %v", rawURL, err)
	}
	request := model.GetRequest("GET", parsed)
	return &request
}
