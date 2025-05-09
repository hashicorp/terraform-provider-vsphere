// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package virtualdevice

import (
	"testing"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

func TestDiskCapacityInGiB(t *testing.T) {
	cases := []struct {
		name     string
		subject  *types.VirtualDisk
		expected int
	}{
		{
			name: "capacityInBytes - integer GiB",
			subject: &types.VirtualDisk{
				CapacityInBytes: 4294967296,
				CapacityInKB:    4194304,
			},
			expected: 4,
		},
		{
			name: "capacityInKB - integer GiB",
			subject: &types.VirtualDisk{
				CapacityInKB: 4194304,
			},
			expected: 4,
		},
		{
			name: "capacityInBytes - non-integer GiB",
			subject: &types.VirtualDisk{
				CapacityInBytes: 4294968320,
				CapacityInKB:    4194305,
			},
			expected: 5,
		},
		{
			name: "capacityInKB - non-integer GiB",
			subject: &types.VirtualDisk{
				CapacityInKB: 4194305,
			},
			expected: 5,
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

func TestFindControllerInfo(t *testing.T) {
	cases := []struct {
		name           string
		deviceList     object.VirtualDeviceList
		disk           *types.VirtualDisk
		expectedUnit   int
		expectedCtlrID int32
		expectError    bool
	}{
		{
			name: "SCSI controller - standard unit",
			deviceList: object.VirtualDeviceList{
				&types.VirtualLsiLogicController{
					VirtualSCSIController: types.VirtualSCSIController{
						VirtualController: types.VirtualController{
							VirtualDevice: types.VirtualDevice{
								Key: 1000,
							},
							BusNumber: 0,
						},
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 1000,
					UnitNumber:    intPtr(5),
				},
			},
			expectedUnit:   5,
			expectedCtlrID: 1000,
			expectError:    false,
		},
		{
			name: "SCSI controller - second bus",
			deviceList: object.VirtualDeviceList{
				&types.VirtualLsiLogicController{
					VirtualSCSIController: types.VirtualSCSIController{
						VirtualController: types.VirtualController{
							VirtualDevice: types.VirtualDevice{
								Key: 1001,
							},
							BusNumber: 1,
						},
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 1001,
					UnitNumber:    intPtr(3),
				},
			},
			expectedUnit:   18, // 15 * 1 + 3
			expectedCtlrID: 1001,
			expectError:    false,
		},
		{
			name: "SATA controller - standard unit",
			deviceList: object.VirtualDeviceList{
				&types.VirtualAHCIController{
					VirtualSATAController: types.VirtualSATAController{
						VirtualController: types.VirtualController{
							VirtualDevice: types.VirtualDevice{
								Key: 15000,
							},
							BusNumber: 0,
						},
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 15000,
					UnitNumber:    intPtr(10),
				},
			},
			expectedUnit:   10,
			expectedCtlrID: 15000,
			expectError:    false,
		},
		{
			name: "SATA controller - second bus",
			deviceList: object.VirtualDeviceList{
				&types.VirtualAHCIController{
					VirtualSATAController: types.VirtualSATAController{
						VirtualController: types.VirtualController{
							VirtualDevice: types.VirtualDevice{
								Key: 15001,
							},
							BusNumber: 1,
						},
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 15001,
					UnitNumber:    intPtr(5),
				},
			},
			expectedUnit:   35, // 30 * 1 + 5
			expectedCtlrID: 15001,
			expectError:    false,
		},
		{
			name: "IDE controller - standard unit",
			deviceList: object.VirtualDeviceList{
				&types.VirtualIDEController{
					VirtualController: types.VirtualController{
						VirtualDevice: types.VirtualDevice{
							Key: 200,
						},
						BusNumber: 0,
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 200,
					UnitNumber:    intPtr(1),
				},
			},
			expectedUnit:   1,
			expectedCtlrID: 200,
			expectError:    false,
		},
		{
			name: "IDE controller - second bus",
			deviceList: object.VirtualDeviceList{
				&types.VirtualIDEController{
					VirtualController: types.VirtualController{
						VirtualDevice: types.VirtualDevice{
							Key: 201,
						},
						BusNumber: 1,
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 201,
					UnitNumber:    intPtr(0),
				},
			},
			expectedUnit:   2, // 2 * 1 + 0
			expectedCtlrID: 201,
			expectError:    false,
		},
		{
			name: "NVMe controller - standard unit",
			deviceList: object.VirtualDeviceList{
				&types.VirtualNVMEController{
					VirtualController: types.VirtualController{
						VirtualDevice: types.VirtualDevice{
							Key: 31000,
						},
						BusNumber: 0,
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 31000,
					UnitNumber:    intPtr(34),
				},
			},
			expectedUnit:   34,
			expectedCtlrID: 31000,
			expectError:    false,
		},
		{
			name: "NVMe controller - second bus",
			deviceList: object.VirtualDeviceList{
				&types.VirtualNVMEController{
					VirtualController: types.VirtualController{
						VirtualDevice: types.VirtualDevice{
							Key: 31001,
						},
						BusNumber: 1,
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 31001,
					UnitNumber:    intPtr(5),
				},
			},
			expectedUnit:   69, // 64 * 1 + 5
			expectedCtlrID: 31001,
			expectError:    false,
		},
		{
			name:       "controller not found",
			deviceList: object.VirtualDeviceList{},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 1000,
					UnitNumber:    intPtr(5),
				},
			},
			expectedUnit:   0,
			expectedCtlrID: 0,
			expectError:    true,
		},
		{
			name: "unit number not set",
			deviceList: object.VirtualDeviceList{
				&types.VirtualLsiLogicController{
					VirtualSCSIController: types.VirtualSCSIController{
						VirtualController: types.VirtualController{
							VirtualDevice: types.VirtualDevice{
								Key: 1000,
							},
							BusNumber: 0,
						},
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 1000,
				},
			},
			expectedUnit:   0,
			expectedCtlrID: 0,
			expectError:    true,
		},
		{
			name: "unsupported controller type",
			deviceList: object.VirtualDeviceList{
				&types.VirtualPCIController{
					VirtualController: types.VirtualController{
						VirtualDevice: types.VirtualDevice{
							Key: 100,
						},
						BusNumber: 0,
					},
				},
			},
			disk: &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					ControllerKey: 100,
					UnitNumber:    intPtr(0),
				},
			},
			expectedUnit:   0,
			expectedCtlrID: 0,
			expectError:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sr := &Subresource{}
			unit, ctlr, err := sr.findControllerInfo(tc.deviceList, tc.disk)

			if tc.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if unit != tc.expectedUnit {
				t.Errorf("expected unit number %d, got %d", tc.expectedUnit, unit)
			}

			if ctlr.GetVirtualController().Key != tc.expectedCtlrID {
				t.Errorf("expected controller ID %d, got %d", tc.expectedCtlrID, ctlr.GetVirtualController().Key)
			}
		})
	}
}

// Helper function to create int pointers
func intPtr(i int32) *int32 {
	return &i
}
