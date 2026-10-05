package ui

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"reflect"
	"strings"
	"sync"

	sprig "github.com/go-task/slim-sprig/v3"

	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
)

const maxPooledRenderBufferSize = 1 << 20

func TemplateFuncMap(
	templates *template.Template,
	assetVersions *AssetVersions,
) template.FuncMap {
	// return map[string]interface{}{}
	fnMap := sprig.GenericFuncMap()

	fnMap["tr"] = func(ctx ctxx.Context, s string) string {
		return wx.T(s).String(ctx)
	}

	fnMap["asset"] = assetVersions.URL

	fnMap["unsafeAttr"] = func(s string) template.HTMLAttr {
		return template.HTMLAttr(s)
	}

	// necessary because default `template` function cannot with
	// dynamic template names;
	// inspired by:
	// https://stackoverflow.com/a/23705598
	//
	// Nested widgets are rendered into a buffer each and copied into their parent, so buffers are
	// reused across renders; growing a fresh buffer per widget dominated allocations of long lists.
	renderBuffers := &sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	}

	// slots instead of slot to make it optional
	fnMap["render"] = func(ctx ctxx.Context, widget any, slots ...string) (template.HTML, error) {
		return renderTemplateWidget(templates, renderBuffers, ctx, widget, slots...)
	}
	return fnMap
}

func renderTemplateWidget(
	templates *template.Template,
	renderBuffers *sync.Pool,
	ctx ctxx.Context,
	widget any,
	slots ...string,
) (template.HTML, error) {
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
	if val.Kind() == reflect.Slice && val.Len() == 0 {
		log.Printf("widgets has no elements, was %T", widget)
		return "", nil
	}

	buf := renderBuffers.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		// don't keep exceptionally large buffers alive
		if buf.Cap() <= maxPooledRenderBufferSize {
			renderBuffers.Put(buf)
		}
	}()

	if val.Kind() == reflect.Slice {
		if err := nilableExecuteWidgetSlice(templates, buf, ctx, val); err != nil {
			return "", err
		}
	} else if err := nilableExecuteWidgetTemplate(templates, buf, ctx, widget); err != nil {
		log.Println(err)
		return "", err
	}

	// copies, the buffer is reused
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

func nilableExecuteWidgetTemplate(
	templates *template.Template,
	buf *bytes.Buffer,
	ctx ctxx.Context,
	widget any,
) error {
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

	if err := templates.ExecuteTemplate(buf, name, widget); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func nilableExecuteWidgetSlice(
	templates *template.Template,
	buf *bytes.Buffer,
	ctx ctxx.Context,
	val reflect.Value,
) error {
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
			if err := nilableExecuteNestedWidgetSlice(templates, buf, ctx, elem); err != nil {
				return err
			}
			continue
		}
		if err := nilableExecuteWidgetTemplate(templates, buf, ctx, qw); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

func nilableExecuteNestedWidgetSlice(
	templates *template.Template,
	buf *bytes.Buffer,
	ctx ctxx.Context,
	val reflect.Value,
) error {
	for i := 0; i < val.Len(); i++ {
		if err := nilableExecuteWidgetTemplate(templates, buf, ctx, val.Index(i).Interface()); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}
