package filesystem

type firstErr struct {
	err error
}

func (qq *firstErr) set(err error) {
	if err != nil && qq.err == nil {
		qq.err = err
	}
}
