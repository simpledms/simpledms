package util

import (
	"encoding/json"
	"html/template"
	"log"
)

/*
// it may be necessary to assign `structs` tags to the converted structs
func JSON(qq ...any) template.JS {
	data := map[string]any{}
	for _, qx := range qq {
		val := structs.Map(qx)
		// merge with data map
		for k, v := range val {
			data[k] = v
		}
	}
	v, err := json.Marshal(data)
	if err != nil {
		log.Panicln(err)
		return ""
	}
	return template.JS(v)
}
*/

// TODO or AsJson or ToJson
// TODO or generic method?
func JSON(qq any) template.JS {
	v, err := json.Marshal(qq)
	if err != nil {
		log.Panicln(err)
		return ""
	}
	return template.JS(v)
}

func JSONStr(qq any) string {
	v, err := json.Marshal(qq)
	if err != nil {
		log.Panicln(err)
		return ""
	}
	return string(v)
}
