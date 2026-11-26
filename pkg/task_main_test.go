package pkg

import (
	"testing"

	"github.com/Qianlitp/crawlergo/pkg/filter"
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

func TestAllReqListContainsOnlyBrowserObservedRequests(t *testing.T) {
	inputTarget := originRequest(t, "https://input.example.com/")
	expansionTarget := originRequest(t, "https://input.example.com/admin/")
	outOfScopeObservation := originRequest(t, "https://cdn.other.example/image.png")

	result := &Result{}
	// Input and path-expansion targets are not observations by themselves.
	if len(result.AllReqList) != 0 {
		t.Fatalf("AllReqList initially contains %d requests, want none", len(result.AllReqList))
	}
	result.recordObservedRequests([]*model.Request{outOfScopeObservation})

	if len(result.AllReqList) != 1 || result.AllReqList[0] != outOfScopeObservation {
		t.Fatalf("AllReqList = %#v, want only the browser-observed request", result.AllReqList)
	}
	if result.AllReqList[0] == inputTarget || result.AllReqList[0] == expansionTarget {
		t.Fatal("an unobserved crawl target was added to AllReqList")
	}
	domains := AllDomainCollect(result.AllReqList)
	if len(domains) != 1 || domains[0] != "cdn.other.example" {
		t.Fatalf("AllDomainCollect(AllReqList) = %#v, want only observed host", domains)
	}
}

func TestInitialTargetStaysInReqListWithoutBrowserObservation(t *testing.T) {
	target := originRequest(t, "https://input.example.com/")
	requestFilter, err := filter.NewScopeFilter([]*model.Request{target}, filter.ScopeFilterCurrDomain)
	if err != nil {
		t.Fatalf("NewScopeFilter() error = %v", err)
	}
	result := &Result{}

	accepted := result.appendInitialTargets([]*model.Request{target}, requestFilter)

	if len(accepted) != 1 || accepted[0] != target {
		t.Fatalf("accepted targets = %#v, want the input target", accepted)
	}
	if len(result.ReqList) != 1 || result.ReqList[0] != target {
		t.Fatalf("ReqList = %#v, want the input target", result.ReqList)
	}
	if len(result.AllReqList) != 0 {
		t.Fatalf("AllReqList = %#v, want no observations", result.AllReqList)
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
