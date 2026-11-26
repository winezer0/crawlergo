package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Qianlitp/crawlergo/pkg/config"
)

func TestCLIWithLocalChromeWebSocket(t *testing.T) {
	websocketURL := localChromeWebSocketURL(t)
	fixture := newCLICompatibilityFixture(t)
	fuzzDictionary := writeFuzzDictionary(t)

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	binaryPath := buildCrawlerBinary(t, repoRoot)
	resultPath := filepath.Join(t.TempDir(), "result.json")
	commandOutput := runCrawlerCLI(t, binaryPath, websocketURL, resultPath, fixture.baseURL, fixture.outOfScopeURL, fuzzDictionary)
	assertCrawlerCLIOutput(t, commandOutput, websocketURL, fixture.baseURL, fixture.outOfScopeURL)
	assertCrawlerResult(t, resultPath)
}

type cliCompatibilityFixture struct {
	baseURL       string
	outOfScopeURL string
}

func newCLICompatibilityFixture(t *testing.T) cliCompatibilityFixture {
	t.Helper()
	outOfScopeServer := startLocalCompatibilityServer(t, "127.0.0.2", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/out-of-scope" {
			_, _ = fmt.Fprint(w, "out of scope")
			return
		}
		http.NotFound(w, r)
	}))
	outOfScopeURL := outOfScopeServer.URL + "/out-of-scope"
	server := startLocalCompatibilityServer(t, "127.0.0.1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, `<html><body><a href="/next?from=home">next</a><script>
fetch('/submit',{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:'field=value'});
fetch(%q);
</script></body></html>`, outOfScopeURL)
		case "/robots.txt":
			_, _ = fmt.Fprint(w, "User-agent: *\nAllow: /from-robots\n")
		case "/from-robots", "/from-fuzz", "/next", "/second", "/submit":
			_, _ = fmt.Fprint(w, "ok")
		default:
			http.NotFound(w, r)
		}
	}))
	return cliCompatibilityFixture{baseURL: server.URL, outOfScopeURL: outOfScopeURL}
}

func startLocalCompatibilityServer(t *testing.T, host string, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	listener, err := net.Listen("tcp4", net.JoinHostPort(host, "0"))
	if err != nil {
		t.Fatalf("listen on local compatibility address %s: %v", host, err)
	}
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return server
}

func localChromeWebSocketURL(t *testing.T) string {
	t.Helper()
	websocketURL := os.Getenv("CRAWLERGO_CHROME_WS_URL")
	if websocketURL == "" {
		t.Skip("set CRAWLERGO_CHROME_WS_URL to run the local Chrome CLI integration test")
	}
	parsed, err := url.Parse(websocketURL)
	if err != nil || (parsed.Scheme != "ws" && parsed.Scheme != "wss") {
		t.Fatal("CRAWLERGO_CHROME_WS_URL must be a valid WebSocket URL")
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		t.Fatal("CRAWLERGO_CHROME_WS_URL must point to a local Chrome instance")
	}
	return websocketURL
}

func writeFuzzDictionary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fuzz-paths.txt")
	if err := os.WriteFile(path, []byte("from-fuzz\n"), 0600); err != nil {
		t.Fatalf("write fuzz dictionary: %v", err)
	}
	return path
}

func buildCrawlerBinary(t *testing.T, repoRoot string) string {
	t.Helper()
	name := "crawlergo"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binaryPath := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/crawlergo")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CrawlerGo CLI: %v\n%s", err, output)
	}
	return binaryPath
}

func runCrawlerCLI(t *testing.T, binaryPath, websocketURL, resultPath, baseURL, outOfScopeURL, fuzzDictionary string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	args := []string{
		"--chrome-ws-url", websocketURL,
		"--output-mode", "none",
		"--output-json", resultPath,
		"--max-crawled-count", "12",
		"--max-run-time", "45",
		"--scope-filter", "curr_domain",
		"--robots-path",
		"--fuzz-path-dict", fuzzDictionary,
		"--log-level", "debug",
		baseURL + "/",
		baseURL + "/second",
	}
	command := exec.CommandContext(ctx, binaryPath, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("CrawlerGo CLI failed: %v\n%s", err, redactTestURLs(string(output), websocketURL, baseURL, outOfScopeURL))
	}
	return string(output)
}

func assertCrawlerCLIOutput(t *testing.T, output, websocketURL string, targetURLs ...string) {
	t.Helper()
	if strings.Contains(output, "unknown IPAddressSpace value") || strings.Contains(output, "unknown ClientNavigationReason value") || strings.Contains(output, "could not unmarshal event") {
		t.Fatalf("local Chrome emitted a CDP decode error:\n%s", redactTestURLs(output, websocketURL, targetURLs...))
	}
	if !strings.Contains(output, "Chrome version Chrome/") || !strings.Contains(output, "CDP protocol 1.3") {
		t.Fatalf("CLI output omitted browser/protocol version:\n%s", redactTestURLs(output, websocketURL, targetURLs...))
	}
}

func assertCrawlerResult(t *testing.T, resultPath string) {
	t.Helper()
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read CrawlerGo result: %v", err)
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode CrawlerGo result: %v", err)
	}

	for _, expected := range []struct {
		method string
		path   string
		source string
		data   string
	}{
		{method: "GET", path: "/", source: config.FromTarget},
		{method: "GET", path: "/second", source: config.FromTarget},
		{method: "GET", path: "/from-robots", source: config.FromRobots},
		{method: "GET", path: "/from-fuzz", source: config.FromFuzz},
	} {
		if !containsCrawlerRequest(result.ReqList, expected.method, expected.path, expected.source, expected.data) {
			t.Errorf("req_list omitted %s %s from %s", expected.method, expected.path, expected.source)
		}
	}
	if !containsCrawlerRequest(result.AllReqList, "POST", "/submit", "", "field=value") {
		t.Errorf("all_req_list omitted the decoded local POST body; observed requests: %s", summarizeCrawlerRequests(result.AllReqList))
	}
	if !containsCrawlerRequest(result.AllReqList, "GET", "/out-of-scope", "", "") {
		t.Error("all_req_list omitted the browser-observed out-of-scope request")
	}
	if containsCrawlerRequest(result.ReqList, "GET", "/out-of-scope", "", "") {
		t.Error("req_list included an out-of-scope request under curr_domain")
	}
}

func summarizeCrawlerRequests(requests []Request) string {
	var summaries []string
	for _, request := range requests {
		parsed, err := url.Parse(request.Url)
		if err != nil {
			continue
		}
		summaries = append(summaries, fmt.Sprintf("%s %s data=%q source=%s", request.Method, parsed.Path, request.Data, request.Source))
	}
	return strings.Join(summaries, "; ")
}

func containsCrawlerRequest(requests []Request, method, path, source, data string) bool {
	for _, request := range requests {
		parsed, err := url.Parse(request.Url)
		if err != nil {
			continue
		}
		if request.Method == method && parsed.Path == path && (source == "" || request.Source == source) && (data == "" || request.Data == data) {
			return true
		}
	}
	return false
}

func redactTestURLs(output, websocketURL string, targetURLs ...string) string {
	for _, rawURL := range append(targetURLs, websocketURL) {
		if rawURL != "" {
			output = strings.ReplaceAll(output, rawURL, "[local test URL redacted]")
		}
	}
	return output
}
