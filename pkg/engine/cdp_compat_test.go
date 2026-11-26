package engine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
)

func TestCDPEnumValuesDecode(t *testing.T) {
	t.Run("IPAddressSpace Loopback", func(t *testing.T) {
		var event network.EventResponseReceivedExtraInfo
		if err := json.Unmarshal([]byte(`{"requestId":"1","resourceIPAddressSpace":"Loopback"}`), &event); err != nil {
			t.Fatalf("decode Loopback: %v", err)
		}
		if event.ResourceIPAddressSpace != network.IPAddressSpaceLoopback {
			t.Fatalf("decoded IPAddressSpace = %q, want %q", event.ResourceIPAddressSpace, network.IPAddressSpaceLoopback)
		}
	})

	for _, value := range []page.ClientNavigationReason{
		page.ClientNavigationReasonInitialFrameNavigation,
		page.ClientNavigationReasonOther,
	} {
		t.Run("ClientNavigationReason "+value.String(), func(t *testing.T) {
			var event page.EventFrameRequestedNavigation
			payload := `{"frameId":"frame-id","reason":"` + value.String() + `","url":"http://127.0.0.1/"}`
			if err := json.Unmarshal([]byte(payload), &event); err != nil {
				t.Fatalf("decode %s: %v", value, err)
			}
			if event.Reason != value {
				t.Fatalf("decoded ClientNavigationReason = %q, want %q", event.Reason, value)
			}
		})
	}
}

func TestRequestPostDataDecodesEntries(t *testing.T) {
	req := &network.Request{
		HasPostData: true,
		PostDataEntries: []*network.PostDataEntry{
			{Bytes: base64.StdEncoding.EncodeToString([]byte("name="))},
			{Bytes: base64.StdEncoding.EncodeToString([]byte("crawlergo"))},
		},
	}
	if got, want := requestPostData(context.Background(), "", req), "name=crawlergo"; got != want {
		t.Fatalf("requestPostData() = %q, want %q", got, want)
	}
}
