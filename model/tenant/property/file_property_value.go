package property

import (
	"math"
	"net/http"
	"strconv"

	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/timex"
)

const maxJSONSafeInteger int64 = 1<<53 - 1

type FilePropertyValue struct {
	typex        fieldtype.FieldType
	text         string
	number       int
	date         timex.Date
	checkbox     bool
}

func NewTextFilePropertyValue(value string) FilePropertyValue {
	return FilePropertyValue{typex: fieldtype.Text, text: value}
}

func NewNumberFilePropertyValue(value int64) (FilePropertyValue, error) {
	number, err := checkedInteger(value)
	return FilePropertyValue{typex: fieldtype.Number, number: number}, err
}

func NewMoneyFilePropertyValue(minorUnits int64) (FilePropertyValue, error) {
	number, err := checkedInteger(minorUnits)
	return FilePropertyValue{typex: fieldtype.Money, number: number}, err
}

func NewDateFilePropertyValue(value timex.Date) (FilePropertyValue, error) {
	if value.IsZero() {
		return FilePropertyValue{}, e.NewHTTPErrorf(http.StatusBadRequest, "Date value is required.")
	}
	return FilePropertyValue{typex: fieldtype.Date, date: value}, nil
}

func NewCheckboxFilePropertyValue(value bool) FilePropertyValue {
	return FilePropertyValue{typex: fieldtype.Checkbox, checkbox: value}
}

func NewDecimalMoneyFilePropertyValue(value float64) (FilePropertyValue, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value > float64(maxJSONSafeInteger)/100 ||
		value < -float64(maxJSONSafeInteger)/100 {
		return FilePropertyValue{}, e.NewHTTPErrorf(http.StatusBadRequest, "Money value is out of range.")
	}
	return NewMoneyFilePropertyValue(int64(math.Round(value * 100)))
}

func checkedInteger(value int64) (int, error) {
	if value < -maxJSONSafeInteger || value > maxJSONSafeInteger ||
		strconv.IntSize == 32 && (value < math.MinInt32 || value > math.MaxInt32) {
		return 0, e.NewHTTPErrorf(http.StatusBadRequest, "Number value is out of range.")
	}
	return int(value), nil
}
