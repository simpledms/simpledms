package dashboard

type RevokeMCPCredentialCmdData struct {
	CredentialID string `validate:"required"`
}
