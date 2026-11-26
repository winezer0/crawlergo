package engine

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	browserproto "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func TestLocalChromeWebSocket(t *testing.T) {
	wsURL := os.Getenv("CRAWLERGO_CHROME_WS_URL")
	if wsURL == "" {
		t.Skip("set CRAWLERGO_CHROME_WS_URL to test a local Chrome WebSocket")
	}
	parsedWSURL, err := url.Parse(wsURL)
	if err != nil || (parsedWSURL.Scheme != "ws" && parsedWSURL.Scheme != "wss") {
		t.Fatal("CRAWLERGO_CHROME_WS_URL must be a valid WebSocket URL")
	}
	wsHost := parsedWSURL.Hostname()
	ip := net.ParseIP(wsHost)
	if wsHost != "localhost" && (ip == nil || !ip.IsLoopback()) {
		t.Fatal("CRAWLERGO_CHROME_WS_URL must point to a local Chrome instance")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fetch" {
			fmt.Fprint(w, "loopback response")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body>local ws page<script>fetch('/fetch',{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:'field=value'})</script></body></html>`)
	}))
	defer server.Close()

	browser := ConnectBrowser(wsURL, nil)
	defer browser.Close()
	chromedpContext := chromedp.FromContext(*browser.Ctx)
	versionCtx := cdp.WithExecutor(*browser.Ctx, chromedpContext.Browser)
	protocolVersion, product, _, _, _, err := browserproto.GetVersion().Do(versionCtx)
	if err != nil {
		t.Fatalf("read local Chrome/CDP version: %v", err)
	}
	t.Logf("local Chrome product=%s, CDP protocol=%s", product, protocolVersion)

	tabCtx, cancel := browser.NewTab(15 * time.Second)
	defer cancel()

	loopbackEvent := make(chan network.IPAddressSpace, 1)
	postBody := make(chan string, 1)
	chromedp.ListenTarget(*tabCtx, func(event interface{}) {
		if event, ok := event.(*network.EventResponseReceivedExtraInfo); ok {
			select {
			case loopbackEvent <- event.ResourceIPAddressSpace:
			default:
			}
		}
		if event, ok := event.(*fetch.EventRequestPaused); ok {
			go func() {
				targetCtx := cdp.WithExecutor(*tabCtx, chromedp.FromContext(*tabCtx).Target)
				if event.Request.Method == "POST" {
					select {
					case postBody <- requestPostData(targetCtx, event.NetworkID, event.Request):
					default:
					}
				}
				_ = fetch.ContinueRequest(event.RequestID).Do(targetCtx)
			}()
		}
	})

	var bodyText string
	if err := chromedp.Run(*tabCtx,
		network.Enable(),
		fetch.Enable(),
		page.Enable(),
		chromedp.Navigate(server.URL),
		chromedp.Evaluate(`document.body.innerText`, &bodyText),
	); err != nil {
		t.Fatalf("navigate through local Chrome WebSocket: %v", err)
	}
	if bodyText != "local ws page" {
		t.Fatalf("page body = %q, want %q", bodyText, "local ws page")
	}
	select {
	case body := <-postBody:
		if body != "field=value" {
			t.Fatalf("decoded request post body = %q, want %q", body, "field=value")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for intercepted POST request")
	}

	select {
	case space := <-loopbackEvent:
		if space != network.IPAddressSpaceLoopback {
			t.Fatalf("response IP address space = %q, want %q", space, network.IPAddressSpaceLoopback)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for local response CDP event")
	}
}
