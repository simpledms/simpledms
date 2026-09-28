package dashboard

type CreateMCPCredentialCmdData struct {
	Label       string `validate:"required,max=100"`
	Destination string `validate:"required"`
	AllowWrites bool
}
