package filter

import (
	"testing"

	"github.com/Qianlitp/crawlergo/pkg/model"
)

func TestCurrDomainScopeUsesAllTargetHosts(t *testing.T) {
	scope, err := NewScopeFilter([]*model.Request{
		scopeRequest(t, "https://shop.example.com/"),
		scopeRequest(t, "https://api.example.net/"),
	}, ScopeFilterCurrDomain)
	if err != nil {
		t.Fatalf("NewScopeFilter() error = %v", err)
	}

	for _, target := range []string{"https://shop.example.com/a", "http://api.example.net/b"} {
		if !scope.InScope(scopeRequest(t, target)) {
			t.Errorf("InScope(%q) = false, want true", target)
		}
	}
	for _, target := range []string{"https://www.example.com/", "https://other.example.org/"} {
		if scope.InScope(scopeRequest(t, target)) {
			t.Errorf("InScope(%q) = true, want false", target)
		}
	}
}

func TestWildDomainScopeUsesAllRegistrableRoots(t *testing.T) {
	scope, err := NewScopeFilter([]*model.Request{
		scopeRequest(t, "https://shop.example.co.uk/"),
		scopeRequest(t, "https://api.example.net/"),
	}, ScopeFilterWildDomain)
	if err != nil {
		t.Fatalf("NewScopeFilter() error = %v", err)
	}

	for _, target := range []string{
		"https://example.co.uk/",
		"https://assets.shop.example.co.uk/",
		"https://other.example.net/",
	} {
		if !scope.InScope(scopeRequest(t, target)) {
			t.Errorf("InScope(%q) = false, want true", target)
		}
	}
	for _, target := range []string{"https://notexample.co.uk/", "https://example.com/"} {
		if scope.InScope(scopeRequest(t, target)) {
			t.Errorf("InScope(%q) = true, want false", target)
		}
	}
}

func TestUnlimitScopeAllowsHTTPAndHTTPSOnly(t *testing.T) {
	scope, err := NewScopeFilter([]*model.Request{scopeRequest(t, "https://example.com/")}, ScopeFilterUnlimit)
	if err != nil {
		t.Fatalf("NewScopeFilter() error = %v", err)
	}

	for _, target := range []string{"https://outside.test/", "http://outside.test/"} {
		if !scope.InScope(scopeRequest(t, target)) {
			t.Errorf("InScope(%q) = false, want true", target)
		}
	}
	if scope.InScope(scopeRequest(t, "ftp://outside.test/file")) {
		t.Fatal("InScope(ftp URL) = true, want false")
	}
}

func TestScopeFilterRejectsInvalidConfigurationAndEnforcesDoFilter(t *testing.T) {
	target := scopeRequest(t, "https://inside.example.com/")
	if _, err := NewScopeFilter([]*model.Request{target}, "unknown"); err == nil {
		t.Fatal("NewScopeFilter() accepted an unknown scope")
	}
	if _, err := NewScopeFilter(nil, ScopeFilterCurrDomain); err == nil {
		t.Fatal("NewScopeFilter() accepted an empty target set")
	}

	scope, err := NewScopeFilter([]*model.Request{target}, ScopeFilterCurrDomain)
	if err != nil {
		t.Fatalf("NewScopeFilter() error = %v", err)
	}
	if scope.DoFilter(target) {
		t.Fatal("DoFilter(target) = true, want target retained")
	}
	if !scope.DoFilter(scopeRequest(t, "https://outside.example.com/")) {
		t.Fatal("DoFilter(out-of-scope request) = false, want request filtered")
	}
}

func scopeRequest(t *testing.T, rawURL string) *model.Request {
	t.Helper()
	parsed, err := model.GetUrl(rawURL)
	if err != nil {
		t.Fatalf("GetUrl(%q) error = %v", rawURL, err)
	}
	request := model.GetRequest("GET", parsed)
	return &request
}
