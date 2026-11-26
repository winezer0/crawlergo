package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func TestExtractResponseURLsFromHTMLOnlyScansInlineScripts(t *testing.T) {
	body := `<html><body><p>text/xml</p><input placeholder="dd/mm/yyyy">
		<script src="/static/app.js"></script>
		<script>const api = "/api/products/v1"; const detail = "/catalogue/item_1.html";
		const mime = "application/x-www-form-urlencoded"; const date = "dd/mm/yyyy";</script>
	</body></html>`

	got := extractResponseURLs(body, "text/html; charset=utf-8")
	want := []string{"/api/products/v1", "/catalogue/item_1.html"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractResponseURLs() = %#v, want %#v", got, want)
	}
}

func TestExtractResponseURLsScansJavaScriptAndJSON(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		body     string
		want     []string
	}{
		{
			name:     "javascript",
			mimeType: "application/javascript",
			body:     `const endpoint = "/api/products/v1"; const mime = "text/plain";`,
			want:     []string{"/api/products/v1"},
		},
		{
			name:     "json",
			mimeType: "application/json",
			body:     `{"endpoint":"/api/products/v1","mime":"application/x-www-form-urlencoded","date":"dd/mm/yyyy"}`,
			want:     []string{"/api/products/v1"},
		},
		{
			name:     "unrelated response type",
			mimeType: "text/css",
			body:     `url = "/api/products/v1"`,
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResponseURLs(tt.body, tt.mimeType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("extractResponseURLs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestShouldParseResponseURLs(t *testing.T) {
	for _, tc := range []struct {
		mimeType string
		want     bool
	}{
		{mimeType: "text/html; charset=utf-8", want: true},
		{mimeType: "application/javascript", want: true},
		{mimeType: "text/javascript", want: true},
		{mimeType: "application/json", want: true},
		{mimeType: "text/css", want: false},
		{mimeType: "font/woff2", want: false},
	} {
		if got := shouldParseResponseURLs(tc.mimeType); got != tc.want {
			t.Errorf("shouldParseResponseURLs(%q) = %t, want %t", tc.mimeType, got, tc.want)
		}
	}
}

func TestIsStaticRequestUsesCDPResourceType(t *testing.T) {
	if !isStaticRequest(network.ResourceTypeFont, "woff%3fv=3.2.1") {
		t.Fatal("font request with an encoded query marker should be static")
	}
	if !isStaticRequest(network.ResourceTypeDocument, "woff") {
		t.Fatal("known static file extension should remain blocked")
	}
	if isStaticRequest(network.ResourceTypeXHR, "") {
		t.Fatal("XHR request without a static extension should remain crawlable")
	}
}

func TestWaitForDOMContentLoaded(t *testing.T) {
	ready := make(chan struct{})
	close(ready)
	if err := waitForDOMContentLoaded(context.Background(), ready); err != nil {
		t.Fatalf("waitForDOMContentLoaded() returned %v for a ready page", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForDOMContentLoaded(ctx, make(chan struct{})); !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForDOMContentLoaded() error = %v, want context.Canceled", err)
	}
}

func TestRunWithTimeOutReturnsDeadline(t *testing.T) {
	action := RunWithTimeOut(nil, 10*time.Millisecond, chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}),
	})
	if err := action.Do(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunWithTimeOut() error = %v, want context.DeadlineExceeded", err)
	}
}
