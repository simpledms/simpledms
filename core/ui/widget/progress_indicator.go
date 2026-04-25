package widget

type ProgressIndicatorType int

const (
	ProgressIndicatorTypeLinear ProgressIndicatorType = iota
	ProgressIndicatorTypeCircular
)

type ProgressIndicatorSize int

const (
	ProgressIndicatorSizeSmall ProgressIndicatorSize = iota
	ProgressIndicatorSizeMedium
	ProgressIndicatorSizeLarge
)

// mainly implemented by Junie
type ProgressIndicator struct {
	Widget[ProgressIndicator]

	// Type defines whether the progress indicator is linear or circular
	Type ProgressIndicatorType

	// Size defines the size of the progress indicator
	Size ProgressIndicatorSize

	// Value represents the current progress value (0-100)
	// If Value is nil, the progress indicator is indeterminate
	Value *float64

	// Color defines the color of the progress indicator
	// If empty, the primary color will be used
	Color string

	IsFixedLayout bool
}

// NewProgressIndicator creates a new progress indicator with default values
func NewProgressIndicator() *ProgressIndicator {
	return &ProgressIndicator{
		Type: ProgressIndicatorTypeLinear,
		Size: ProgressIndicatorSizeMedium,
	}
}

// Linear sets the progress indicator type to linear
func (qq *ProgressIndicator) Linear() *ProgressIndicator {
	qq.Type = ProgressIndicatorTypeLinear
	return qq
}

// Circular sets the progress indicator type to circular
func (qq *ProgressIndicator) Circular() *ProgressIndicator {
	qq.Type = ProgressIndicatorTypeCircular
	return qq
}

// Small sets the progress indicator size to small
func (qq *ProgressIndicator) Small() *ProgressIndicator {
	qq.Size = ProgressIndicatorSizeSmall
	return qq
}

// Medium sets the progress indicator size to medium
func (qq *ProgressIndicator) Medium() *ProgressIndicator {
	qq.Size = ProgressIndicatorSizeMedium
	return qq
}

// Large sets the progress indicator size to large
func (qq *ProgressIndicator) Large() *ProgressIndicator {
	qq.Size = ProgressIndicatorSizeLarge
	return qq
}

// SetValue sets the progress value (0-100)
// If value is nil, the progress indicator becomes indeterminate
func (qq *ProgressIndicator) SetValue(value *float64) *ProgressIndicator {
	qq.Value = value
	return qq
}

// SetColor sets the color of the progress indicator
func (qq *ProgressIndicator) SetColor(color string) *ProgressIndicator {
	qq.Color = color
	return qq
}

// IsLinear returns true if the progress indicator is linear
func (qq *ProgressIndicator) IsLinear() bool {
	return qq.Type == ProgressIndicatorTypeLinear
}

// IsCircular returns true if the progress indicator is circular
func (qq *ProgressIndicator) IsCircular() bool {
	return qq.Type == ProgressIndicatorTypeCircular
}

// IsSmall returns true if the progress indicator is small
func (qq *ProgressIndicator) IsSmall() bool {
	return qq.Size == ProgressIndicatorSizeSmall
}

// IsMedium returns true if the progress indicator is medium
func (qq *ProgressIndicator) IsMedium() bool {
	return qq.Size == ProgressIndicatorSizeMedium
}

// IsLarge returns true if the progress indicator is large
func (qq *ProgressIndicator) IsLarge() bool {
	return qq.Size == ProgressIndicatorSizeLarge
}

// IsIndeterminate returns true if the progress indicator is indeterminate
func (qq *ProgressIndicator) IsIndeterminate() bool {
	return qq.Value == nil
}

// GetValue returns the current progress value or 0 if indeterminate
func (qq *ProgressIndicator) GetValue() float64 {
	if qq.Value == nil {
		return 0
	}
	return *qq.Value
}

// GetColorClass returns the color class for the progress indicator
func (qq *ProgressIndicator) GetColorClass() string {
	if qq.Color != "" {
		return qq.Color
	}
	return "bg-primary"
}
