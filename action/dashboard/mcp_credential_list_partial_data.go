package dashboard

type MCPCredentialListPartialData struct {
	Destination            string
	CredentialStatusValues []string `url:"credential_status,omitempty"`
}

func (qq *MCPCredentialListPartialData) statusFilter() (bool, bool, error) {
	return parseCredentialStatusFilter(qq.CredentialStatusValues)
}
