package dashboard

type MCPCredentialListData struct {
	Offset int `validate:"min=0,max=1000000"`
}
