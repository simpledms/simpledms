package dashboard

type RevokeMCPCredentialCmdData struct {
	CredentialPublicID string `validate:"required"`
}
