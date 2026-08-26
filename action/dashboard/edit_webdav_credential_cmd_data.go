package dashboard

type EditWebDAVCredentialCmdData struct {
	CredentialPublicID string `validate:"required" form_attr_type:"hidden"`
	DeviceLabel        string `validate:"required" form_attrs:"autofocus"`
}
