package dashboard

type CreateWebDAVCredentialCmdData struct {
	Destination       string `validate:"required"`
	Label             string `validate:"required"`
	SecretLength      *int   `json:",omitempty" validate:"omitempty,min=12,max=43"`
	CompatibilityMode bool   `json:",omitempty"`
}
