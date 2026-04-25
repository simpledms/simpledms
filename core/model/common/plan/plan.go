//go:generate go tool enumer -type=Plan -sql -ent -json -empty_string -output=plan.gen.go
package plan

type Plan int

const (
	Unknown Plan = iota
	Trial
	Pro
	Unlimited
)
