package managetenantusers

type RevokeWebDAVCredentialCmdData struct {
	CredentialPublicID string `validate:"required"`
}
