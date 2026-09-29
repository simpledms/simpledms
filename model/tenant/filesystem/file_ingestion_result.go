package filesystem

// FileIngestionResult identifies a successfully committed file ingestion.
type FileIngestionResult struct {
	FilePublicID string
	Filename     string
	Size         int64
	IsInInbox    bool
}
