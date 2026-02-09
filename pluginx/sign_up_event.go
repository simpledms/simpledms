package pluginx

type SignUpEvent struct {
	AccountID             int64
	AccountPublicID       string
	AccountEmail          string
	TenantID              int64
	TenantPublicID        string
	TenantName            string
	SubscribeToNewsletter bool
}
