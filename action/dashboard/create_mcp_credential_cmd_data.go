package dashboard

type CreateMCPCredentialCmdData struct {
	Destination string `validate:"required"`
	Label       string `validate:"required,max=100"`
	AllowWrites bool   `json:",omitempty"`
}
