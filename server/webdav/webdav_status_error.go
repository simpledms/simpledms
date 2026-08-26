package webdav

type webDAVStatusError struct {
	status int
	msg    string
}

func (qq webDAVStatusError) Error() string { return qq.msg }
