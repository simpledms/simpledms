package filesystem

type progressWriter struct {
	total int64
}

func (pw *progressWriter) Read(p []byte) (n int, err error) {
	pw.total += int64(len(p))
	// log.Printf("Uploaded %d bytes so far\n", pw.total)
	return len(p), nil
}
