package scheduler

import (
	"context"
	"time"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/db/entmain/session"
)

func (qq *Scheduler) cleanupSessions() {
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
