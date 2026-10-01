package dashboard

type WebDAVCredentialListPartialData struct {
	Destination            string
	CredentialStatusValues []string `url:"credential_status,omitempty"`
}

func (qq *WebDAVCredentialListPartialData) statusFilter() (bool, bool, error) {
	return parseCredentialStatusFilter(qq.CredentialStatusValues)
}
