package dashboard

import (
	"net/http"

	"github.com/simpledms/simpledms/util/e"
)

const (
	webDAVCredentialStatusActive  = "active"
	webDAVCredentialStatusRevoked = "revoked"
)

type WebDAVCredentialListPartialData struct {
	Destination            string
	CredentialStatusValues []string `url:"credential_status,omitempty"`
}

func (qq *WebDAVCredentialListPartialData) statusFilter() (bool, bool, error) {
	if len(qq.CredentialStatusValues) == 0 {
		return true, false, nil
	}

	var showActive bool
	var showRevoked bool
	for _, status := range qq.CredentialStatusValues {
		switch status {
		case webDAVCredentialStatusActive:
			showActive = true
		case webDAVCredentialStatusRevoked:
			showRevoked = true
		default:
			return false, false, e.NewHTTPErrorf(
				http.StatusBadRequest,
				"Form validation failed.",
			)
		}
	}
	return showActive, showRevoked, nil
}
