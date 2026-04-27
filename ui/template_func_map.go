package ui

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"reflect"
	"strings"

	sprig "github.com/go-task/slim-sprig/v3"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	wx "github.com/simpledms/simpledms/ui/widget"
)

func TemplateFuncMap(templates *template.Template) template.FuncMap {
	// return map[string]interface{}{}
	fnMap := sprig.GenericFuncMap()

	fnMap["tr"] = func(ctx ctxx.Context, s string) string {
		return wx.T(s).String(ctx)
	}

	fnMap["unsafeAttr"] = func(s string) template.HTMLAttr {
		return template.HTMLAttr(s)
	}

	// necessary because default `template` function cannot with
	// dynamic template names;
	// inspired by:
	// https://stackoverflow.com/a/23705598
	//
	// slots instead of slot to make it optional
	fnMap["render"] = func(ctx ctxx.Context, widget any, slots ...string) (template.HTML, error) {
		// need reflection because it is not possible to match any slice in signature
		// just a specific slice
		// TODO is this fast enough?
		val := reflect.ValueOf(widget)

		// if val.Kind() == reflect.Struct && val.IsZero() {
		// = elem.Addr()
		// }

		// val.IsNil is necessary because `any` is never `nil`;
		// IsZero might be safer than IsNil because IsNil can panic if type is not supported,
		// but IsZero may have side effects because in same cases struct with zero value should
		// still be rendered
		//
		// added Kind = Pointer check on 08.09.24 to make handling of empty structs possible,
		// they would otherwise panic on val.IsNil() because a struct cannot be nil
		//
		// log.Println(val, widget)
		// log.Println(val.IsValid())
		// log.Println(val.IsNil())
		// log.Println(val, widget)
		if widget == nil || (val.Kind() == reflect.Pointer && val.IsNil()) {
			// log.Printf("widget is nil, was %T", widget)
			return "", nil
		}

		buf := bytes.NewBuffer([]byte{})

		executeTemplate := func(buf *bytes.Buffer, widget any) error {
			// TODO is this correct?
			if widgetx, isRenderable := widget.(renderable.Renderable); isRenderable {
				widgetx.SetContext(ctx)
				if ctx == nil {
					log.Printf("ctx is nil was %T, %+v", widgetx, widgetx)
				}
			} else {
				log.Printf("widget doesn't implement renderable.Renderable, was %T", widget)
			}

			name := fmt.Sprintf("%T", widget)
			name = strings.TrimPrefix(name, "widget.") // for embed HTMXAttrs
			name = strings.TrimPrefix(name, "*widget.")

			// necessary for partials that embed a container but have
			// no html template file
			if named, ok := widget.(Named); ok {
				name = named.Name()
			}

			// log.Println(name)

			err := templates.ExecuteTemplate(buf, name, widget)
			if err != nil {
				log.Println(err)
				return err
			}

			return nil
		}

		switch val.Kind() {
		case reflect.Slice:
			if val.Len() == 0 {
				log.Printf("widgets has no elements, was %T", widget)
				return "", nil
			}
			for i := 0; i < val.Len(); i++ {
				qw := val.Index(i).Interface()
				elem := val.Index(i).Elem()

				// .IsNil cannot be called on struct because struct cannot be nil,
				// and it panics if done anyway, thus use pointer to struct for
				// further processing;
				// happend for example when handling []*Tab on *TabBar, but it worked fine before
				// when []IWidget got used instead of []*Tab
				//
				// disabled on 09.08.24 and impl Kind = Pointer check below instead
				/* if elem.Kind() == reflect.Struct {
					elem = elem.Addr()
				}*/

				// handle nil values in slices; makes creation of views simpler;
				// for example if a back button should only be shown conditionally, nil
				// can be added if the condition is not met and it's not necessary to
				// create a slice declaration before the view definition
				//
				// log.Printf("%T", widget)
				// log.Println(elem, qw, elem.Type(), elem.Kind())
				// log.Println(elem.IsNil())
				// log.Printf("%T", widget)
				if elem.Kind() == reflect.Pointer && elem.IsNil() {
					log.Println(elem)
					continue
				}

				// TODO quick workaround for one layer child slices, implement more robust recursive solution!
				//		was implement for groupChips in filterTagsModal
				if elem.Kind() == reflect.Slice {
					for j := 0; j < elem.Len(); j++ {
						err := executeTemplate(buf, elem.Index(j).Interface())
						if err != nil {
							log.Println(err)
							return "", err
						}
					}
					continue
				}

				err := executeTemplate(buf, qw)
				if err != nil {
					log.Println(err)
					return "", err
				}
			}
		default: // struct
			err := executeTemplate(buf, widget)
			if err != nil {
				log.Println(err)
				return "", err
			}
		}

		htmlStr := buf.String()

		if len(slots) > 0 {
			htmlStr = fmt.Sprintf(
				"<div slot=\"%s\">%s</div>",
				slots[0],
				htmlStr,
			)
		}

		return template.HTML(htmlStr), nil
	}
	return fnMap
}
