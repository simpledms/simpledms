package sqlx

import (
	"os"
)

// Development mode logs every SQL query. Setting this to true turns the log off, for example
// for e2e runs, where it otherwise writes tens of megabytes per run.
const devSQLLogDisabledEnv = "SIMPLEDMS_DEV_DISABLE_SQL_LOG"

func isDevSQLLogDisabled() bool {
	return os.Getenv(devSQLLogDisabledEnv) == "true"
}
