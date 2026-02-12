package filesystem

import "testing"

func TestStorageQuotaSkipsValidationWhenSaaSDisabled(t *testing.T) {
	storageQuota := NewStorageQuota(false)

	err := storageQuota.EnsureTenantStorageLimit(nil, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStorageQuotaSkipsValidationForNonPositiveIncomingSize(t *testing.T) {
	storageQuota := NewStorageQuota(true)

	err := storageQuota.EnsureTenantStorageLimit(nil, 0)
	if err != nil {
		t.Fatalf("expected no error for zero bytes, got %v", err)
	}

	err = storageQuota.EnsureTenantStorageLimit(nil, -1)
	if err != nil {
		t.Fatalf("expected no error for negative bytes, got %v", err)
	}
}

func TestStorageQuotaExceedsStorageLimit(t *testing.T) {
	storageQuota := NewStorageQuota(true)

	tests := []struct {
		name                 string
		usedBytes            int64
		incomingUploadedSize int64
		limitBytes           int64
		expected             bool
	}{
		{name: "non-positive incoming size", usedBytes: 5, incomingUploadedSize: 0, limitBytes: 10, expected: false},
		{name: "zero limit", usedBytes: 0, incomingUploadedSize: 1, limitBytes: 0, expected: true},
		{name: "already at limit", usedBytes: 10, incomingUploadedSize: 1, limitBytes: 10, expected: true},
		{name: "already above limit", usedBytes: 11, incomingUploadedSize: 1, limitBytes: 10, expected: true},
		{name: "fits exactly into remaining", usedBytes: 9, incomingUploadedSize: 1, limitBytes: 10, expected: false},
		{name: "exceeds remaining", usedBytes: 9, incomingUploadedSize: 2, limitBytes: 10, expected: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual := storageQuota.exceedsStorageLimit(tt.usedBytes, tt.incomingUploadedSize, tt.limitBytes)
			if actual != tt.expected {
				t.Fatalf(
					"expected %v for used=%d incoming=%d limit=%d, got %v",
					tt.expected,
					tt.usedBytes,
					tt.incomingUploadedSize,
					tt.limitBytes,
					actual,
				)
			}
		})
	}
}
