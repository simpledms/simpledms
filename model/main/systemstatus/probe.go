package systemstatus

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
)

// probe runs a connection check in the background, so that all services are checked in
// parallel and the page waits at most for the slowest one.
type probe struct {
	done   chan struct{}
	detail string
	err    error
}

func startProbe(ctx context.Context, check func(context.Context) (string, error)) *probe {
	probex := &probe{
		done: make(chan struct{}),
	}

	go func() {
		defer close(probex.done)
		// the goroutine is outside the request recovery of Router.wrapTx; a panicking client
		// must not crash the server
		defer func() {
			if r := recover(); r != nil {
				log.Printf("%v: %s", r, debug.Stack())
				probex.err = fmt.Errorf("check failed: %v", r)
			}
		}()

		probex.detail, probex.err = check(ctx)
		if probex.err != nil {
			log.Println(probex.err)
		}
	}()

	return probex
}

// wait returns the detail, for example the version of the service, or the error of the check.
func (qq *probe) wait() (string, error) {
	<-qq.done
	return qq.detail, qq.err
}
