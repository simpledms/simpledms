package dashboard

type RevokeWebDAVCredentialCmdData struct {
	CredentialPublicID string `validate:"required"`
	TenantID           int64
}
