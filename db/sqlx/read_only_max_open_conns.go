package sqlx

import (
	"log"
	"os"
	"runtime"
	"strconv"
)

// used only for testing
const dbReadOnlyMaxOpenConnsEnv = "SIMPLEDMS_DB_READ_ONLY_MAX_OPEN_CONNS"

func readOnlyMaxOpenConns() int {
	if envValue := os.Getenv(dbReadOnlyMaxOpenConnsEnv); envValue != "" {
		parsedValue, err := strconv.Atoi(envValue)
		if err != nil {
			log.Fatalf("invalid %s: %v", dbReadOnlyMaxOpenConnsEnv, err)
		}
		if parsedValue < 1 {
			log.Fatalf("%s must be at least 1", dbReadOnlyMaxOpenConnsEnv)
		}
		return parsedValue
	}

	numCPU := runtime.NumCPU()
	if numCPU < 2 {
		// 2 is a good minimum even for 1 CPU thread because it reduces the risk of a deadlock,
		// for example if the scheduler is running.
		//
		// only for production/dev use because env value is just used for testing
		// and it should be possible to simulate just one reader
		return 2
	}
	return numCPU
}
