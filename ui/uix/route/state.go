package route

import (
	"fmt"
	"log"
	"slices"

	"github.com/go-playground/form"
)

func WrapWithState(fn func(string, string, string) string, data any) func(string, string, string) string {
	return func(tenantID, spaceID, fileID string) string {
		return WithState(fn, data)(tenantID, spaceID, fileID)
	}
}

// currently only used for spaces
func RootWithState2(fn func() string, data any) string {
	if data == nil {
		return fn()
	}

	// values := url.Values{}
	// encoder := schema.NewEncoder()
	// encoder.SetAliasTag("url")

	encoder := form.NewEncoder()
	encoder.SetTagName("url")

	values, err := encoder.Encode(data)
	if err != nil {
		log.Println(err)
		return fn()
	}

	if len(values) == 0 {
		return fn()
	}

	// TODO use same method as in actionx.Common to get normalized URL?
	return fmt.Sprintf("%s?%s", fn(), values.Encode())
}

func RootWithState(fn func(string, string) string, data any) func(string, string) string {
	return func(tenantID, spaceID string) string {
		if data == nil {
			return fn(tenantID, spaceID)
		}

		// values := url.Values{}
		// encoder := schema.NewEncoder()
		// encoder.SetAliasTag("url")

		encoder := form.NewEncoder()
		encoder.SetTagName("url")

		values, err := encoder.Encode(data)
		if err != nil {
			log.Println(err)
			return fn(tenantID, spaceID)
		}

		if len(values) == 0 {
			return fn(tenantID, spaceID)
		}

		// TODO use same method as in actionx.Common to get normalized URL?
		return fmt.Sprintf("%s?%s", fn(tenantID, spaceID), values.Encode())
	}
}

// FIXME safety?
func WithState(fn func(string, string, string) string, data any) func(string, string, string) string {
	return func(tenantID, spaceID, fileID string) string {
		if data == nil {
			return fn(tenantID, spaceID, fileID)
		}

		// values := url.Values{}
		// encoder := schema.NewEncoder()
		// encoder.SetAliasTag("url")

		encoder := form.NewEncoder()
		encoder.SetTagName("url")

		values, err := encoder.Encode(data)
		if err != nil {
			log.Println(err)
			return fn(tenantID, spaceID, fileID)
		}

		if len(values) == 0 {
			return fn(tenantID, spaceID, fileID)
		}

		// remove duplicate values from slices
		//
		// custom encoder doesn't work because signature just allows to return one string value
		// and not a slice of strings
		for k, v := range values {
			slices.Sort(v)
			v = slices.Compact(v)
			// v = slices.Clip(v) // TODO necessary? seems not
			values[k] = v
		}

		// TODO use same method as in actionx.Common to get normalized URL?
		return fmt.Sprintf("%s?%s", fn(tenantID, spaceID, fileID), values.Encode())
	}
}

func WithState2(fn func(string, string, string, string) string, data any) func(string, string, string, string) string {
	return func(tenantID, spaceID, dirID, fileID string) string {
		if data == nil {
			return fn(tenantID, spaceID, dirID, fileID)
		}

		// values := url.Values{}
		// encoder := schema.NewEncoder()
		// encoder.SetAliasTag("url")
		encoder := form.NewEncoder()
		encoder.SetTagName("url")

		values, err := encoder.Encode(data)
		if err != nil {
			log.Println(err)
			return fn(tenantID, spaceID, dirID, fileID)
		}

		if len(values) == 0 {
			return fn(tenantID, spaceID, dirID, fileID)
		}

		// TODO use same method as in actionx.Common to get normalized URL?
		return fmt.Sprintf("%s?%s", fn(tenantID, spaceID, dirID, fileID), values.Encode())
	}
}
