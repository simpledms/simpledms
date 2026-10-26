package cookiex

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/simpledms/simpledms/util/httpx"
)

func TestSessionCookieSameSiteMatchesSecureMode(t *testing.T) {
	tests := []struct {
		name     string
		insecure bool
		want     http.SameSite
	}{
		{name: "secure", want: http.SameSiteNoneMode},
		{name: "insecure development", insecure: true, want: http.SameSiteLaxMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("issued", func(t *testing.T) {
				rr := httptest.NewRecorder()
				cookie, err := SetSessionCookie(
					httpx.NewResponseWriter(rr),
					httpx.NewRequest(httptest.NewRequest(http.MethodPost, "/", nil)),
					false,
					tt.insecure,
				)
				if err != nil {
					t.Fatal(err)
				}
				assertSessionCookie(t, cookie, !tt.insecure, tt.want)
				assertSetCookieHeader(t, rr, cookie)
			})

			t.Run("renewed", func(t *testing.T) {
				rr := httptest.NewRecorder()
				cookie, renewed := RenewSessionCookie(
					httpx.NewResponseWriter(rr), "session", time.Now().Add(24*time.Hour), tt.insecure,
				)
				if !renewed {
					t.Fatal("expected session cookie renewal")
				}
				assertSessionCookie(t, cookie, !tt.insecure, tt.want)
				if cookie.Value != "session" {
					t.Errorf("expected session value to be preserved, got %q", cookie.Value)
				}
				now := time.Now()
				if cookie.Expires.Before(now.Add(13*24*time.Hour)) ||
					cookie.Expires.After(now.Add(15*24*time.Hour)) {
					t.Errorf("renewed expiry outside expected bounds: %v", cookie.Expires)
				}
				assertSetCookieHeader(t, rr, cookie)
			})

			t.Run("deleted", func(t *testing.T) {
				rr := httptest.NewRecorder()
				InvalidateSessionCookie(httpx.NewResponseWriter(rr), tt.insecure)
				var cookie *http.Cookie
				for _, responseCookie := range rr.Result().Cookies() {
					if responseCookie.Name == SessionCookieName() {
						cookie = responseCookie
						break
					}
				}
				assertSessionCookie(t, cookie, !tt.insecure, tt.want)
				if cookie.MaxAge != -1 {
					t.Errorf("expected deletion MaxAge -1, got %d", cookie.MaxAge)
				}
			})
		})
	}
}

func TestRenewSessionCookieRefreshesPolicyWithoutChangingFreshExpiry(t *testing.T) {
	expiresAt := time.Now().Add(14 * 24 * time.Hour)
	for _, tt := range []struct {
		name     string
		insecure bool
		want     http.SameSite
	}{
		{name: "secure", want: http.SameSiteNoneMode},
		{name: "insecure development", insecure: true, want: http.SameSiteLaxMode},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			cookie, renewed := RenewSessionCookie(
				httpx.NewResponseWriter(rr), "session", expiresAt, tt.insecure,
			)
			if renewed {
				t.Fatal("expected policy refresh without extending fresh session")
			}
			assertSessionCookie(t, cookie, !tt.insecure, tt.want)
			if !cookie.Expires.Equal(expiresAt) {
				t.Errorf("expected expiry %v, got %v", expiresAt, cookie.Expires)
			}
			assertSetCookieHeader(t, rr, cookie)
		})
	}
}

func TestRenewSessionCookieRefreshesTemporarySessionPolicy(t *testing.T) {
	rr := httptest.NewRecorder()
	cookie, renewed := RenewSessionCookie(
		httpx.NewResponseWriter(rr), "session", time.Time{}, false,
	)
	if renewed {
		t.Fatal("expected temporary session policy refresh without expiry renewal")
	}
	assertSessionCookie(t, cookie, true, http.SameSiteNoneMode)
	if !cookie.Expires.IsZero() {
		t.Errorf("expected session cookie expiry to remain zero, got %v", cookie.Expires)
	}
	assertSetCookieHeader(t, rr, cookie)
}

func assertSetCookieHeader(t *testing.T, rr *httptest.ResponseRecorder, cookie *http.Cookie) {
	t.Helper()
	for _, responseCookie := range rr.Result().Cookies() {
		if responseCookie.Name == cookie.Name {
			if responseCookie.String() != cookie.String() {
				t.Errorf("Set-Cookie does not match returned cookie: got %q, want %q", responseCookie, cookie)
			}
			return
		}
	}
	t.Errorf("no Set-Cookie header for %q", cookie.Name)
}

func assertSessionCookie(t *testing.T, cookie *http.Cookie, secure bool, sameSite http.SameSite) {
	t.Helper()
	if cookie == nil {
		t.Fatal("expected session cookie")
	}
	if cookie.Secure != secure || !cookie.HttpOnly || cookie.Path != "/" ||
		cookie.SameSite != sameSite {
		t.Errorf("unexpected session cookie attributes: %+v", cookie)
	}
}
