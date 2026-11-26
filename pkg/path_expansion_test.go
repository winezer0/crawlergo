package pkg

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Qianlitp/crawlergo/pkg/config"
	"github.com/Qianlitp/crawlergo/pkg/model"
)

func TestDoFuzzSubmitsEveryPathWithinBoundedConcurrency(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		if request.URL.Path == "/found" {
			response.WriteHeader(http.StatusOK)
			return
		}
		response.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	parsedURL, err := model.GetUrl(server.URL)
	if err != nil {
		t.Fatalf("GetUrl() error = %v", err)
	}
	navRequest := model.GetRequest(config.GET, parsedURL)
	paths := make([]string, 100)
	for index := range paths {
		paths[index] = fmt.Sprintf("missing-%03d", index)
	}
	paths[len(paths)-1] = "found"
	paths = append(paths, "/found\r\n")

	results := doFuzz(navRequest, paths)
	if got := atomic.LoadInt32(&requestCount); got != int32(len(paths)) {
		t.Fatalf("fuzz sent %d requests, want all %d dictionary paths", got, len(paths))
	}
	if len(results) != 1 {
		t.Fatalf("doFuzz() returned %d results, want one", len(results))
	}
	if got := results[0]; got.URL.Host != parsedURL.Host || got.URL.Path != "/found" || got.Source != config.FromFuzz {
		t.Fatalf("fuzz result = %#v", got)
	}
}
