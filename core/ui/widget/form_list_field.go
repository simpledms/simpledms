package widget

import (
	"log"
	"reflect"

	"github.com/marcobeierer/structs"
)

type formListField struct {
	Name string
}

type formListFields []*formListField

func newFormListFields(obj interface{}) formListFields {
	listFields := formListFields{}

	for _, structField := range structs.Fields(obj) {
		if elementTag := structField.Tag("element"); elementTag == "onebyone" {
			value := structField.Value()

			switch reflect.TypeOf(value).Kind() {
			case reflect.Slice:
				slice := reflect.ValueOf(value)

				for qi := 0; qi < slice.Len(); qi++ {
					listField := &formListField{
						Name: slice.Index(qi).String(),
					}
					listFields = append(listFields, listField)
				}
			case reflect.Map:
				mapx := reflect.ValueOf(value)

				// for qi := 0; qi < mapx.Len(); qi++ {
				for _, value := range mapx.MapKeys() {
					listField := &formListField{
						Name: value.String(),
					}
					listFields = append(listFields, listField)
				}
			default:
				log.Println("no case for kind", reflect.TypeOf(value).Kind())
			}
		} else if structField.IsEmbedded() {
			listFields = append(listFields, newFormListFields(structField.Value())...)
		} else {
			listField := &formListField{
				Name: structField.Name(),
			}
			listFields = append(listFields, listField)
		}
	}

	return listFields
}
