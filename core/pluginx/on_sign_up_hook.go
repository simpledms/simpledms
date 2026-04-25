package pluginx

import "github.com/simpledms/simpledms/ctxx"

type OnSignUpHook interface {
	OnSignUp(ctx ctxx.Context, event SignUpEvent) error
}
