package widget

type GridSize int

const (
	GridSizeDefault GridSize = iota
	GridSizeSmall
	GridSizeMedium
	GridSizeLarge
)

// TODO rename to CardGrid?
type Grid struct {
	Widget[Grid]
	HTMXAttrs

	// TODO not a good approach to define number of columns,
	Size GridSize
	// Rows    int

	Heading  *Heading
	Children IWidget
	Footer   IWidget // TODO good name?
}

func (qq *Grid) GetColumnsClass() string {
	switch qq.Size {
	case GridSizeDefault:
		fallthrough
	default:
		return "grid-cols-1 md:grid-cols-2 exp:grid-cols-3 lg:grid-cols-4"
	}
	/*
		// full class name required for purging CSS
		switch qq.Columns {
		case 1:
			return "grid-cols-1"
		case 2:
			return "grid-cols-2"
		case 3:
			return "grid-cols-3"
		case 4:
			return "grid-cols-1 md:grid-cols-2 exp:grid-cols-3 lg:grid-cols-4"
		case 5:
			return "grid-cols-5"
		case 6:
			return "grid-cols-6"
		case 7:
			return "grid-cols-7"
		case 8:
			return "grid-cols-8"
		case 9:
			return "grid-cols-9"
		case 10:
			return "grid-cols-10"
		case 11:
			return "grid-cols-11"
		case 12:
			return "grid-cols-12"
		default:
			return "grid-cols-1"
		}
	*/
}

/*
func (qq *Grid) GetRowsClass() string {
	switch qq.Rows {
	case 1:
		return "grid-rows-1"
	case 2:
		return "grid-rows-2"
	case 3:
		return "grid-rows-3"
	case 4:
		return "grid-rows-4"
	case 5:
		return "grid-rows-5"
	case 6:
		return "grid-rows-6"
	case 7:
		return "grid-rows-7"
	case 8:
		return "grid-rows-8"
	case 9:
		return "grid-rows-9"
	case 10:
		return "grid-rows-10"
	case 11:
		return "grid-rows-11"
	case 12:
		return "grid-rows-12"
	default:
		return "grid-rows-1"
	}
}
*/
