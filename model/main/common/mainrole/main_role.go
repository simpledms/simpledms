//go:generate go tool enumer -type=MainRole -sql -ent -json -empty_string -output=main_role.gen.go
package mainrole

type MainRole int

const (
	User MainRole = iota + 1
	Supporter
	Admin
)
