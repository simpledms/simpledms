package scheduler

import (
	"context"
	"log"
	"runtime/debug"
	"time"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/app/simpledms/entmain/session"
)

func (qq *Scheduler) cleanupSessions() {
	defer func() {
		// tested and works
		if r := recover(); r != nil {
			log.Printf("%v: %s", r, debug.Stack())
			log.Println("trying to recover")

			// TODO what is a good interval
			time.Sleep(5 * time.Minute)

			// tested and works, automatically restarts loop
			qq.cleanupSessions()
		}
	}()

	ctx := context.Background()
	ctx = privacy.DecisionContext(ctx, privacy.Allow)

	// delete normal session
	qq.mainDB.ReadWriteConn.Session.Delete().Where(
		// have 5 minutes buffer to not accidentally kill a session or get a conc conflict if
		// a user signs out in the meantime;
		// longer values could be a security risk if the expiration date is not checked
		// when validating the session; the check is implement, so a short value is just
		// another security layer
		session.DeletableAtLT(time.Now().Add(-5 * time.Minute)),
	).ExecX(ctx)
}
