package property

import (
	"math"
	"strconv"
	"testing"

	"github.com/simpledms/simpledms/model/main/common/fieldtype"
)

func TestIntegerFilePropertyValueConstructors(t *testing.T) {
	maxInt := int64(math.MaxInt)
	minInt := int64(math.MinInt)
	maxValue := min(maxJSONSafeInteger, maxInt)
	minValue := max(-maxJSONSafeInteger, minInt)

	constructors := []struct {
		name  string
		call  func(int64) (FilePropertyValue, error)
		typex fieldtype.FieldType
	}{
		{
			name:  "number",
			call:  NewNumberFilePropertyValue,
			typex: fieldtype.Number,
		},
		{
			name:  "money",
			call:  NewMoneyFilePropertyValue,
			typex: fieldtype.Money,
		},
	}

	// Integer values must fit both native int and the JSON-safe +/- (1<<53-1) range.
	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			type testCase struct {
				name    string
				value   int64
				want    int
				wantErr bool
			}
			tests := []testCase{
				{
					name:  "lower intersection boundary",
					value: minValue,
					want:  int(minValue),
				},
				{
					name:  "upper intersection boundary",
					value: maxValue,
					want:  int(maxValue),
				},
				{
					name:  "zero",
					value: 0,
					want:  0,
				},
				{
					name:  "negative",
					value: -1,
					want:  -1,
				},
				{
					name:    "below JSON safe range",
					value:   -maxJSONSafeInteger - 1,
					wantErr: true,
				},
				{
					name:    "above JSON safe range",
					value:   maxJSONSafeInteger + 1,
					wantErr: true,
				},
				{
					name:    "minimum int64",
					value:   math.MinInt64,
					wantErr: true,
				},
				{
					name:    "maximum int64",
					value:   math.MaxInt64,
					wantErr: true,
				},
			}
			if strconv.IntSize == 32 {
				tests = append(tests,
					testCase{
						name:    "below native int",
						value:   minInt - 1,
						wantErr: true,
					},
					testCase{
						name:    "above native int",
						value:   maxInt + 1,
						wantErr: true,
					},
				)
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					value, err := constructor.call(test.value)
					if (err != nil) != test.wantErr {
						t.Fatalf("error = %v, wantErr %t", err, test.wantErr)
					}
					if test.wantErr {
						return
					}
					if value.number != test.want {
						t.Errorf("number = %d, want %d", value.number, test.want)
					}
					if value.typex != constructor.typex {
						t.Errorf("type = %v, want %v", value.typex, constructor.typex)
					}
				})
			}
		})
	}
}
