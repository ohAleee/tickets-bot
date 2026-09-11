package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TicketsBot-cloud/archiverclient"
	"github.com/TicketsBot-cloud/dashboard/config"
	"github.com/TicketsBot-cloud/gdl/objects/channel"
	v2 "github.com/TicketsBot-cloud/logarchiver/pkg/model/v2"
	"github.com/gin-gonic/gin"
)

func testContext() *gin.Context {
	config.Conf.Server.Secret = "test-secret"

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "http://dash.example.com/api/1/transcripts/2/render", nil)
	return ctx
}

func TestSignedUrlRoundTrip(t *testing.T) {
	messages := []v2.Message{{
		Attachments: []channel.Attachment{
			{Id: 999, Filename: "a b.png", Url: archiverclient.MediaPath(1, 2, 999, "a b.png"), ProxyUrl: archiverclient.MediaPath(1, 2, 999, "a b.png")},
			{Id: 1000, Filename: "big.zip", Url: "https://cdn.discordapp.com/attachments/big.zip?ex=1"},
		},
	}}

	SignAttachmentUrls(testContext(), 1, 2, messages)

	signed := messages[0].Attachments[0].Url
	if !strings.HasPrefix(signed, "http://dash.example.com/media/1/2/999/a%20b.png?exp=") {
		t.Fatalf("unexpected signed url: %s", signed)
	}

	if messages[0].Attachments[0].ProxyUrl != signed {
		t.Fatal("proxy_url was not signed")
	}

	// Not mirrored: must be left alone
	if messages[0].Attachments[1].Url != "https://cdn.discordapp.com/attachments/big.zip?ex=1" {
		t.Fatal("unmirrored attachment was rewritten")
	}

	// The handler verifies the signature over the decoded filename
	exp, sig, _ := strings.Cut(strings.SplitN(signed, "?exp=", 2)[1], "&sig=")
	if got := mediaSignature(1, 2, 999, "a b.png", mustAtoi64(t, exp)); got != sig {
		t.Fatalf("signature mismatch: %s != %s", got, sig)
	}

	if mediaSignature(1, 2, 999, "other.png", mustAtoi64(t, exp)) == sig {
		t.Fatal("signature does not cover the filename")
	}
}

func mustAtoi64(t *testing.T, s string) int64 {
	t.Helper()

	var out int64
	for _, c := range s {
		if c < '0' || c > '9' {
			t.Fatalf("not a number: %s", s)
		}
		out = out*10 + int64(c-'0')
	}

	return out
}
