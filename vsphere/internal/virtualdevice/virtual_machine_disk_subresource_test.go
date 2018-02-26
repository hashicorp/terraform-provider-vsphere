package virtualdevice

import (
	"testing"

	"github.com/vmware/govmomi/vim25/types"
)

func TestDiskCapacityInGiB(t *testing.T) {
	cases := []struct {
		name     string
		subject  *types.VirtualDisk
		expected int
	}{
		{
			name: "capacityInBytes",
			subject: &types.VirtualDisk{
				CapacityInBytes: 4294967296,
				CapacityInKB:    4194304,
			},
			expected: 4,
		},
		{
			name: "capacityInKB",
			subject: &types.VirtualDisk{
				CapacityInKB: 4194304,
			},
			expected: 4,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := diskCapacityInGiB(tc.subject)
			if tc.expected != actual {
				t.Fatalf("expected %d, got %d", tc.expected, actual)
			}
		})
	}
}
