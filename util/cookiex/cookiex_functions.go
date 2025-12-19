package cookiex

import (
	"log"
	"net/http"
	"time"

	gonanoid "github.com/matoous/go-nanoid"

	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

// TODO find a better location, but is kind of a path...
func SessionCookieName() string {
	return "simpledms_session"
}

// IMPORTANT
// caller must add cookie to database
func SetSessionCookie(rw httpx.ResponseWriter, req *httpx.Request, isTemporarySession bool) (*http.Cookie, error) {
	// TODO check that no session exists for this user yet? what about mobile and desktop?
	//		probably not a good idea

	cookie, err := req.Cookie(SessionCookieName())
	if err == nil {
		// only return if cookie is set and is valid
		if errx := cookie.Valid(); errx == nil {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Auth cookie already set.")
		}
	}

	// TODO what is a good length?
	sessionID, err := gonanoid.Generate("_-0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", 64)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not generate session id.")
	}

	// expires instead of max age because with maxAge it is not possible
	// to find out how to long the cookie is valid which is important for renewal
	expires := time.Time{} // session based
	if !isTemporarySession {
		expires = time.Now().Add(time.Hour * 24 * 14) // 2 weeks
	}

	// duplicate in RenewSessionCookie
	cookie = &http.Cookie{
		Name:   SessionCookieName(),
		Value:  sessionID,
		Quoted: false, // TODO?

		Path:    "/",
		Domain:  "",
		Expires: expires,

		MaxAge:   0,
		Secure:   true,
		HttpOnly: true,
		// shouldn't be necessary with http.CrossOriginProtection middleware
		// but works as a fallback for older browsers because SameSite is longer
		// in baseline than Sec-Fetch-Site; only works for same site, not same origin
		SameSite: http.SameSiteLaxMode, // TODO is Lax enough or do we need Strict?
	}
	http.SetCookie(rw, cookie)

	return cookie, nil
}

// IMPORTANT
// caller must remove cookie from database
func InvalidateSessionCookie(rw httpx.ResponseWriter) {
	// IMPORTANT if changed, cookie creation in SignIn has to be changed too
	http.SetCookie(rw, &http.Cookie{
		Name:     SessionCookieName(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // deletes cookie
		HttpOnly: true,
		Secure:   true,
		// shouldn't be necessary with http.CrossOriginProtection middleware
		// but works as a fallback for older browsers because SameSite is longer
		// in baseline than Sec-Fetch-Site; only works for same site, not same origin
		SameSite: http.SameSiteLaxMode, // TODO is Lax enough or do we need Strict?
	})
}

// IMPORTANT
// caller must update session in db, so that it doesn't get deleted by scheduler
//
// only value is available when reading a cookie from a request;
// thus renewal cookie must be reconstructed;
// expiresAt must be read from database
func RenewSessionCookie(rw httpx.ResponseWriter, value string, expiresAt time.Time) (*http.Cookie, bool) {
	if expiresAt.IsZero() {
		// temporary session cookie
		return nil, false
	}

	// renew cookie if it expires in less than 13 days
	if expiresAt.After(time.Now().Add(time.Hour * 24 * 13)) {
		// valid more than 13 days
		return nil, false
	}

	// duplicate in SetSessionCookie
	cookie := &http.Cookie{
		Name:   SessionCookieName(),
		Value:  value,
		Quoted: false, // TODO?

		Path:    "/",
		Domain:  "",
		Expires: time.Now().Add(time.Hour * 24 * 14),

		MaxAge:   0,
		Secure:   true,
		HttpOnly: true,
		// shouldn't be necessary with http.CrossOriginProtection middleware
		// but works as a fallback for older browsers because SameSite is longer
		// in baseline than Sec-Fetch-Site; only works for same site, not same origin
		SameSite: http.SameSiteLaxMode, // TODO is Lax enough or do we need Strict?
	}

	http.SetCookie(rw, cookie)
	return cookie, true
}

func DeletableAt(cookie *http.Cookie) time.Time {
	if cookie.Expires.IsZero() {
		// kill temporary sessions after 12 hours
		// TODO what is a good value? should be communicated in login form
		return time.Now().Add(time.Hour * 12)
	}
	return cookie.Expires
}
