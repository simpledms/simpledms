package actionx

// TODO is form prefix necessary?
type ResponseWrapper string

const (
	ResponseWrapperNone   ResponseWrapper = ""
	ResponseWrapperDialog ResponseWrapper = "dialog"
	// ResponseWrapperDialog  ResponseWrapper = "dialog"
)

func (qq ResponseWrapper) String() string {
	return string(qq)
}
