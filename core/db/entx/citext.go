package entx

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// use custom type instead of COLLATE NOCASE because the latter only works
// for ASCII letters, not all Unicode chars

type CIText string

func NewCIText(s string) CIText {
	return CIText(strings.ToLower(s))
}

func (qq CIText) Value() (driver.Value, error) {
	// TODO is this good enough or unicode.ToLower necessary?
	return qq.String(), nil
}

func (qq *CIText) Scan(value interface{}) error {
	val, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan to CIText, expected string, got %T, value was %v", value, value)
	}
	*qq = CIText(val)
	return nil
}

func (qq CIText) String() string {
	return strings.ToLower(string(qq))
}
