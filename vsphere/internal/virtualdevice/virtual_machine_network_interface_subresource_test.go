package virtualdevice

import (
	"testing"

	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

func TestNicUnitRange(t *testing.T) {
	cases := []struct {
		name     string
		devices  object.VirtualDeviceList
		expected int
	}{
		{
			name: "basic",
			devices: object.VirtualDeviceList{
				&types.VirtualVmxnet3{
					VirtualVmxnet: types.VirtualVmxnet{
						VirtualEthernetCard: types.VirtualEthernetCard{
							VirtualDevice: types.VirtualDevice{
								UnitNumber: structure.Int32Ptr(7),
							},
						},
					},
				},
				&types.VirtualE1000{
					VirtualEthernetCard: types.VirtualEthernetCard{
						VirtualDevice: types.VirtualDevice{
							UnitNumber: structure.Int32Ptr(8),
						},
					},
				},
				&types.VirtualE1000{
					VirtualEthernetCard: types.VirtualEthernetCard{
						VirtualDevice: types.VirtualDevice{
							UnitNumber: structure.Int32Ptr(9),
						},
					},
				},
			},
			expected: 3,
		},
		{
			name: "single NIC not at first offset",
			devices: object.VirtualDeviceList{
				&types.VirtualVmxnet3{
					VirtualVmxnet: types.VirtualVmxnet{
						VirtualEthernetCard: types.VirtualEthernetCard{
							VirtualDevice: types.VirtualDevice{
								UnitNumber: structure.Int32Ptr(8),
							},
						},
					},
				},
			},
			expected: 2,
		},
		{
			name: "hole in middle",
			devices: object.VirtualDeviceList{
				&types.VirtualVmxnet3{
					VirtualVmxnet: types.VirtualVmxnet{
						VirtualEthernetCard: types.VirtualEthernetCard{
							VirtualDevice: types.VirtualDevice{
								UnitNumber: structure.Int32Ptr(7),
							},
						},
					},
				},
				&types.VirtualE1000{
					VirtualEthernetCard: types.VirtualEthernetCard{
						VirtualDevice: types.VirtualDevice{
							UnitNumber: structure.Int32Ptr(9),
						},
					},
				},
			},
			expected: 3,
		},
		{
			name: "hole in middle and not starting at first offset",
			devices: object.VirtualDeviceList{
				&types.VirtualVmxnet3{
					VirtualVmxnet: types.VirtualVmxnet{
						VirtualEthernetCard: types.VirtualEthernetCard{
							VirtualDevice: types.VirtualDevice{
								UnitNumber: structure.Int32Ptr(8),
							},
						},
					},
				},
				&types.VirtualE1000{
					VirtualEthernetCard: types.VirtualEthernetCard{
						VirtualDevice: types.VirtualDevice{
							UnitNumber: structure.Int32Ptr(10),
						},
					},
				},
			},
			expected: 4,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := nicUnitRange(tc.devices)
			if err != nil {
				t.Fatalf("bad: %s", err)
			}
			if tc.expected != actual {
				t.Fatalf("expected %d, got %d", tc.expected, actual)
			}
		})
	}
}
