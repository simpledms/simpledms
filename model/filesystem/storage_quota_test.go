package filesystem

import (
	"math"
	"testing"

	"github.com/simpledms/simpledms/model/common/plan"
)

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

func TestStorageQuotaLimitBytesForPlan(t *testing.T) {
	storageQuota := NewStorageQuota(true)

	tests := []struct {
		name            string
		tenantPlan      plan.Plan
		activeUserCount int
		expected        int64
	}{
		{
			name:            "trial plan fixed limit",
			tenantPlan:      plan.Trial,
			activeUserCount: 0,
			expected:        tenantQuotaTrialBytes,
		},
		{
			name:            "trial ignores user count",
			tenantPlan:      plan.Trial,
			activeUserCount: 7,
			expected:        tenantQuotaTrialBytes,
		},
		{
			name:            "pro per active user",
			tenantPlan:      plan.Pro,
			activeUserCount: 2,
			expected:        2 * tenantQuotaProPerUserBytes,
		},
		{
			name:            "pro no active users",
			tenantPlan:      plan.Pro,
			activeUserCount: 0,
			expected:        0,
		},
		{
			name:            "unlimited fixed limit",
			tenantPlan:      plan.Unlimited,
			activeUserCount: 0,
			expected:        tenantQuotaUnlimitedBytes,
		},
		{
			name:            "unknown defaults to trial model",
			tenantPlan:      plan.Unknown,
			activeUserCount: 3,
			expected:        tenantQuotaTrialBytes,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual := storageQuota.limitBytesForPlan(tt.tenantPlan, tt.activeUserCount)
			if actual != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, actual)
			}
		})
	}
}

func TestStorageQuotaLimitBytesForPlanCapsAtMaxInt64(t *testing.T) {
	storageQuota := NewStorageQuota(true)

	overflowActiveUsers := int(math.MaxInt64/tenantQuotaProPerUserBytes) + 1
	actual := storageQuota.limitBytesForPlan(plan.Pro, overflowActiveUsers)
	if actual != math.MaxInt64 {
		t.Fatalf("expected %d, got %d", int64(math.MaxInt64), actual)
	}
}

func TestStorageQuotaExceedsStorageLimitAcrossPlans(t *testing.T) {
	storageQuota := NewStorageQuota(true)

	tests := []struct {
		name               string
		tenantPlan         plan.Plan
		activeUserCount    int
		usedBytes          int64
		incomingUploadSize int64
		expected           bool
	}{
		{
			name:               "unknown below trial limit",
			tenantPlan:         plan.Unknown,
			activeUserCount:    10,
			usedBytes:          tenantQuotaTrialBytes - 1,
			incomingUploadSize: 1,
			expected:           false,
		},
		{
			name:               "unknown above trial limit",
			tenantPlan:         plan.Unknown,
			activeUserCount:    10,
			usedBytes:          tenantQuotaTrialBytes,
			incomingUploadSize: 1,
			expected:           true,
		},
		{
			name:               "trial below limit",
			tenantPlan:         plan.Trial,
			activeUserCount:    10,
			usedBytes:          0,
			incomingUploadSize: 1,
			expected:           false,
		},
		{
			name:               "trial exactly at limit",
			tenantPlan:         plan.Trial,
			activeUserCount:    10,
			usedBytes:          tenantQuotaTrialBytes - 1,
			incomingUploadSize: 1,
			expected:           false,
		},
		{
			name:               "trial above limit",
			tenantPlan:         plan.Trial,
			activeUserCount:    10,
			usedBytes:          tenantQuotaTrialBytes,
			incomingUploadSize: 1,
			expected:           true,
		},
		{
			name:               "pro below limit",
			tenantPlan:         plan.Pro,
			activeUserCount:    2,
			usedBytes:          0,
			incomingUploadSize: 1,
			expected:           false,
		},
		{
			name:               "pro exactly at limit",
			tenantPlan:         plan.Pro,
			activeUserCount:    2,
			usedBytes:          2*tenantQuotaProPerUserBytes - 1,
			incomingUploadSize: 1,
			expected:           false,
		},
		{
			name:               "pro above limit",
			tenantPlan:         plan.Pro,
			activeUserCount:    2,
			usedBytes:          2 * tenantQuotaProPerUserBytes,
			incomingUploadSize: 1,
			expected:           true,
		},
		{
			name:               "unlimited below limit",
			tenantPlan:         plan.Unlimited,
			activeUserCount:    0,
			usedBytes:          0,
			incomingUploadSize: 1,
			expected:           false,
		},
		{
			name:               "unlimited exactly at limit",
			tenantPlan:         plan.Unlimited,
			activeUserCount:    0,
			usedBytes:          tenantQuotaUnlimitedBytes - 1,
			incomingUploadSize: 1,
			expected:           false,
		},
		{
			name:               "unlimited above limit",
			tenantPlan:         plan.Unlimited,
			activeUserCount:    0,
			usedBytes:          tenantQuotaUnlimitedBytes,
			incomingUploadSize: 1,
			expected:           true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			limitBytes := storageQuota.limitBytesForPlan(tt.tenantPlan, tt.activeUserCount)
			actual := storageQuota.exceedsStorageLimit(
				tt.usedBytes,
				tt.incomingUploadSize,
				limitBytes,
			)
			if actual != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}
