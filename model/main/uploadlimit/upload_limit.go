package uploadlimit

import (
	"fmt"
	"math"
	"net/http"

	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/fileutil"
)

const bytesPerMiB int64 = 1024 * 1024

type UploadLimit struct {
	maxUploadSizeMib int64
}

func NewUploadLimitFromMiB(maxUploadSizeMib int64) (*UploadLimit, error) {
	if maxUploadSizeMib < 0 {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Max upload size must be greater than or equal to 0 MiB.")
	}
	if maxUploadSizeMib > math.MaxInt64/bytesPerMiB {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Max upload size is too large.")
	}

	return &UploadLimit{
		maxUploadSizeMib: maxUploadSizeMib,
	}, nil
}

func NewUploadLimitFromBytes(maxUploadSizeBytes int64) (*UploadLimit, error) {
	if maxUploadSizeBytes < 0 {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Max upload size must be greater than or equal to 0.")
	}

	maxUploadSizeMib := int64(0)
	if maxUploadSizeBytes > 0 {
		maxUploadSizeMib = maxUploadSizeBytes / bytesPerMiB
		if maxUploadSizeBytes%bytesPerMiB != 0 {
			maxUploadSizeMib++
		}
	}

	return NewUploadLimitFromMiB(maxUploadSizeMib)
}

func NewUploadLimitFromForm(isUnlimited bool, maxUploadSizeMib int64) (*UploadLimit, error) {
	if isUnlimited {
		return NewUploadLimitFromMiB(0)
	}
	if maxUploadSizeMib <= 0 {
		return nil, e.NewHTTPErrorf(
			http.StatusBadRequest,
			"Max upload size must be greater than 0 MiB when unlimited is disabled.",
		)
	}

	return NewUploadLimitFromMiB(maxUploadSizeMib)
}

func (qq *UploadLimit) IsUnlimited() bool {
	return qq.maxUploadSizeMib <= 0
}

func (qq *UploadLimit) MiB() int64 {
	return qq.maxUploadSizeMib
}

func (qq *UploadLimit) Bytes() int64 {
	if qq.IsUnlimited() {
		return 0
	}

	return qq.maxUploadSizeMib * bytesPerMiB
}

func (qq *UploadLimit) MiBLabel() string {
	return fmt.Sprintf("%d MiB", qq.maxUploadSizeMib)
}

func (qq *UploadLimit) LabelWithUnlimited(unlimitedLabel string) string {
	if qq.IsUnlimited() {
		return unlimitedLabel
	}

	return fileutil.FormatSize(qq.Bytes()) + " (" + qq.MiBLabel() + ")"
}
