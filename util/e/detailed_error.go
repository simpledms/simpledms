package e

/*
import (
	"time"
)

type Context map[string]interface{}

// TODO required for structural logging
type DetailedError struct {
	Error
	UserID    int64
	Severity  loglevel.LogLevel // TODO custom enum with None, Info, Warning, Error, maybe Critical
	Subject   string // TODO rename to SubjectType?
	SubjectID int64  // for example companyID
	UserIP    string
	Action    string
	Context   Context `structs:",value"`
	Date      time.Time
}

/*
func NewDetailedError() {
}

func NewDetailedErrorWithContext() {

}
*/
