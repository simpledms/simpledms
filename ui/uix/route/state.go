package route

import (
	"fmt"
	"log"
	"net/url"
	"slices"

	"github.com/go-playground/form"
)

func RootWithState(fn func(string, string) string, data any) func(string, string) string {
	return func(tenantID, spaceID string) string {
		return withQueryState(fn(tenantID, spaceID), data, false)
	}
}

// FIXME safety?
func WithState(fn func(string, string, string) string, data any) func(string, string, string) string {
	return func(tenantID, spaceID, fileID string) string {
		return withQueryState(fn(tenantID, spaceID, fileID), data, true)
	}
}

func WithState2(fn func(string, string, string, string) string, data any) func(string, string, string, string) string {
	return func(tenantID, spaceID, dirID, fileID string) string {
		return withQueryState(fn(tenantID, spaceID, dirID, fileID), data, false)
	}
}

// withQueryState appends data, encoded via its `url` struct tags, as query to path.
// It returns path unchanged if data is nil, empty, or cannot be encoded.
func withQueryState(path string, data any, isDeduplicatingValues bool) string {
	if data == nil {
		return path
	}

	// values := url.Values{}
	// encoder := schema.NewEncoder()
	// encoder.SetAliasTag("url")

	encoder := form.NewEncoder()
	encoder.SetTagName("url")

	values, err := encoder.Encode(data)
	if err != nil {
		log.Println(err)
		return path
	}

	if len(values) == 0 {
		return path
	}

	if isDeduplicatingValues {
		removeDuplicateValues(values)
	}

	// TODO use same method as in actionx.Common to get normalized URL?
	return fmt.Sprintf("%s?%s", path, values.Encode())
}

// remove duplicate values from slices
//
// custom encoder doesn't work because signature just allows to return one string value
// and not a slice of strings
func removeDuplicateValues(values url.Values) {
	for k, v := range values {
		slices.Sort(v)
		v = slices.Compact(v)
		// v = slices.Clip(v) // TODO necessary? seems not
		values[k] = v
	}
}
