//go:generate go tool enumer -type=TenantRole -sql -ent -json -empty_string -output=tenant_role.gen.go
package tenantrole

type TenantRole int

const (
	User TenantRole = iota + 1
	Owner
	// TODO full access, but cannot view files?
	// Supporter
)
