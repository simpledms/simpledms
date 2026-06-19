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

	return runtime.NumCPU()
}
