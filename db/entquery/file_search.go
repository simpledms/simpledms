package entquery

import (
	"strings"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filesearch"
	"github.com/simpledms/simpledms/db/enttenant/predicate"
)

const fileSearchCandidateLimit = 1000

// FileIsDirectory filters files with integer equality so SQLite can use composite indexes.
func FileIsDirectory(value bool) predicate.File {
	return fileBoolEQ(file.FieldIsDirectory, value)
}

// FileIsInInbox filters files with integer equality so SQLite can use composite indexes.
func FileIsInInbox(value bool) predicate.File {
	return fileBoolEQ(file.FieldIsInInbox, value)
}

func fileBoolEQ(field string, value bool) predicate.File {
	intValue := 0
	if value {
		intValue = 1
	}

	return func(qs *sql.Selector) {
		qs.Where(sql.EQ(qs.C(field), intValue))
	}
}

// ApplyFileSearchCandidateFilter limits a file query to an FTS candidate set.
func ApplyFileSearchCandidateFilter(
	qs *sql.Selector,
	searchQuery string,
	spaceID int64,
	isInInbox bool,
) {
	applyFileSearchCandidateFilter(qs, searchQuery, spaceID, isInInbox, false, false)
}

// ApplyFileSearchCandidateFilterWithDirectory limits a file query to an FTS candidate set.
func ApplyFileSearchCandidateFilterWithDirectory(
	qs *sql.Selector,
	searchQuery string,
	spaceID int64,
	isInInbox bool,
	isDirectory bool,
) {
	applyFileSearchCandidateFilter(qs, searchQuery, spaceID, isInInbox, true, isDirectory)
}

func applyFileSearchCandidateFilter(
	qs *sql.Selector,
	searchQuery string,
	spaceID int64,
	isInInbox bool,
	shouldFilterDirectory bool,
	isDirectory bool,
) {
	fileSearchTable := sql.Table(filesearch.Table)

	qs.Where(sql.In(
		qs.C(file.FieldID),
		fileSearchCandidateQuery(
			fileSearchTable,
			searchQuery,
			spaceID,
			isInInbox,
			shouldFilterDirectory,
			isDirectory,
		),
	))
}

func fileSearchCandidateQuery(
	fileSearchTable *sql.SelectTable,
	searchQuery string,
	spaceID int64,
	isInInbox bool,
	shouldFilterDirectory bool,
	isDirectory bool,
) *sql.Selector {
	predicates := []*sql.Predicate{
		sql.EQ(fileSearchTable.C(filesearch.FieldFileSearches), searchQuery),
		sql.EQ(fileSearchTable.C(file.FieldSpaceID), spaceID),
		sql.EQ(fileSearchTable.C(file.FieldIsInInbox), boolInt(isInInbox)),
	}
	if shouldFilterDirectory {
		predicates = append(
			predicates,
			sql.EQ(fileSearchTable.C(file.FieldIsDirectory), boolInt(isDirectory)),
		)
	}

	candidateQuery := sql.Select(fileSearchTable.C(filesearch.FieldRowid)).
		From(fileSearchTable).
		Where(sql.And(predicates...))

	// LIMIT protects broad natural-language searches, but it is much slower for
	// exact filename-like phrase searches on large FTS5 indexes.
	if shouldLimitFileSearchCandidates(searchQuery) {
		candidateQuery = candidateQuery.Limit(fileSearchCandidateLimit)
	}

	return candidateQuery
}

func shouldLimitFileSearchCandidates(searchQuery string) bool {
	return len(searchQuery) <= 24 && !strings.ContainsAny(searchQuery, `.-_/\\`)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
