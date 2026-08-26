package dashboard

type CreateWebDAVCredentialCmdData struct {
	Destination string `validate:"required"`
	Label       string `validate:"required"`
}
