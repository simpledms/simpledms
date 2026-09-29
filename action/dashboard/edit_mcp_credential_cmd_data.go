package dashboard

type EditMCPCredentialCmdData struct {
	CredentialPublicID string `validate:"required" form_attr_type:"hidden"`
	ClientLabel        string `validate:"required,max=100" form_attrs:"autofocus"`
}
