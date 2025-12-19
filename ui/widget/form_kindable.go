package widget

import "reflect"

type formKindable interface {
	Kind() reflect.Kind
}
