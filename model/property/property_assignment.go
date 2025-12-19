package property

import (
	"fmt"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/model/common/fieldtype"
)

// TODO correct location?
type PropertyAssignment struct {
	Data *enttenant.FilePropertyAssignment
}

func NewPropertyAssignment(data *enttenant.FilePropertyAssignment) *PropertyAssignment {
	return &PropertyAssignment{data}
}

func (qq *PropertyAssignment) String(ctx ctxx.Context, propertym *Property) string {
	// TODO propertym as argument or inject into PropertyAssignment? this way we don't need
	//	 	eager loading, caller can decide
	//		but could fell apart if multiple indirections... can this happen with small aggregates?
	//		probably a domainservice would be created if necessary...

	if propertym.Data.ID != qq.Data.PropertyID {
		panic("property assignment not for this property")
	}

	switch propertym.Data.Type {
	case fieldtype.Text:
		return qq.Data.TextValue
	case fieldtype.Number:
		return fmt.Sprintf("%d", qq.Data.NumberValue)
	case fieldtype.Money:
		// TODO okay?
		return fmt.Sprintf("%.2f", float64(qq.Data.NumberValue)/100.0)
	case fieldtype.Checkbox:
		return fmt.Sprintf("%s", propertym.Data.Name)
	case fieldtype.Date:
		// TODO format: for displaying we may want user date format, for filenames year first...
		return fmt.Sprintf("%s", qq.Data.DateValue.String(""))
	default:
		// TODO okay?
		panic("unknown property type")
	}
}
