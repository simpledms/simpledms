package auth

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/util/httpx"
)

func TestDecodeJSONBodySecurityLimits(t *testing.T) {
	t.Run("accepts normal JSON", func(t *testing.T) {
		req := httpx.NewRequest(httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"ok"}`)))
		var got struct {
			Name string `json:"name"`
		}
		if err := decodeJSONBody(req, &got); err != nil || got.Name != "ok" {
			t.Fatalf("decodeJSONBody() = %+v, %v", got, err)
		}
	})

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "oversized trailing whitespace", body: `{"name":"ok"}` + strings.Repeat(" ", 1024*1024)},
		{name: "trailing JSON value", body: `{"name":"ok"} {}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httpx.NewRequest(httptest.NewRequest("POST", "/", strings.NewReader(tc.body)))
			var got struct {
				Name string `json:"name"`
			}
			if err := decodeJSONBody(req, &got); err == nil {
				t.Fatal("decodeJSONBody() accepted invalid request body")
			}
		})
	}
}
