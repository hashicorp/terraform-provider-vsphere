// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package license

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vmware/govmomi/license"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
)

// MaskLicenseKey returns a masked version of the license key for secure logging.
// Format: First 4 chars + ****** + last 4 chars
func MaskLicenseKey(key string) string {
	if len(key) < 9 {
		return "********"
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

// MaskedLicenseKeyLogOperation logs license operations with masked key information
func MaskedLicenseKeyLogOperation(ctx context.Context, operation string, key string, additional map[string]interface{}) {
	if additional == nil {
		additional = make(map[string]interface{})
	}

	additional["masked_key"] = MaskLicenseKey(key)
	additional["key_length"] = len(key)

	tflog.Debug(ctx, "License operation: "+operation, additional)
}

// GetLicenseInfoFromKey retrieves license information based on the provided license key using the license manager.
func GetLicenseInfoFromKey(ctx context.Context, key string, manager *license.Manager) *types.LicenseManagerLicenseInfo {
	tflog.Debug(ctx, "Attempting to get license info")

	tflog.Debug(ctx, "Listing all licenses via license manager")
	infoList, err := manager.List(ctx)
	if err != nil {
		tflog.Error(ctx, "Failed to list licenses from vSphere", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}

	tflog.Debug(ctx, "Iterating through license list to find match", map[string]interface{}{
		"listSize": len(infoList),
	})
	for i := range infoList {
		info := infoList[i]
		if info.LicenseKey == key {
			tflog.Debug(ctx, "Found matching license key in list")
			return &info
		}
	}

	tflog.Debug(ctx, "License key not found in the list")
	return nil
}

// KeyExists checks if a given license key exists within the license manager.
func KeyExists(ctx context.Context, key string, manager *license.Manager) bool {
	tflog.Debug(ctx, "Checking if license key exists")
	tflog.Debug(ctx, "Listing all licenses via license manager to check existence")
	infoList, err := manager.List(ctx)
	if err != nil {
		tflog.Error(ctx, "Failed to list licenses while checking key existence", map[string]interface{}{
			"error": err.Error(),
		})
		return false
	}

	tflog.Debug(ctx, "Iterating through license list to find key", map[string]interface{}{
		"listSize": len(infoList),
	})
	for _, info := range infoList {
		if info.LicenseKey == key {
			tflog.Debug(ctx, "Found matching license key")
			return true
		}
	}

	tflog.Debug(ctx, "License key not found in the list")
	return false
}

// UpdateLabels updates labels for a specified license key using the provided label map.
func UpdateLabels(ctx context.Context, manager *license.Manager, licenseKey string, labelMap map[string]interface{}) error {
	tflog.Debug(ctx, "Updating labels for a specific license resource", map[string]interface{}{
		"labelCount": len(labelMap),
	})

	for key, value := range labelMap {
		stringValue, ok := value.(string)
		if !ok {
			err := fmt.Errorf("label value for key '%s' is not a string (type: %T)", key, value)
			tflog.Error(ctx, "Invalid label value type during update", map[string]interface{}{
				"labelKey":  key,
				"valueType": fmt.Sprintf("%T", value),
				"error":     err.Error(),
			})
			return err
		}

		tflog.Debug(ctx, "Updating individual license label", map[string]interface{}{
			"labelKey":   key,
			"labelValue": stringValue,
		})

		err := UpdateLabel(ctx, manager, licenseKey, key, stringValue)
		if err != nil {
			tflog.Error(ctx, "Failed to update individual license label", map[string]interface{}{
				"labelKey":   key,
				"labelValue": stringValue,
				"error":      err.Error(),
			})
			return fmt.Errorf("failed to update label '%s' for the license resource: %w", key, err)
		}
	}

	tflog.Debug(ctx, "Successfully updated all labels for the license resource")
	return nil
}

// UpdateLabel assigns or updates the specified label key-value pair for a given license using the license manager.
func UpdateLabel(ctx context.Context, m *license.Manager, licenseKey string, key string, val string) error {
	tflog.Debug(ctx, "Attempting to update a single license label", map[string]interface{}{
		"labelKey":   key,
		"labelValue": val,
	})

	req := types.UpdateLicenseLabel{
		This:       m.Reference(),
		LicenseKey: licenseKey,
		LabelKey:   key,
		LabelValue: val,
	}

	_, err := methods.UpdateLicenseLabel(ctx, m.Client(), &req)
	if err != nil {
		tflog.Error(ctx, "Failed API call to update license label", map[string]interface{}{
			"labelKey":   key,
			"labelValue": val,
			"error":      err.Error(),
		})
		return fmt.Errorf("failed to update label '%s': %w", key, err)
	}

	tflog.Debug(ctx, "Successfully updated single license label via API", map[string]interface{}{
		"labelKey":   key,
		"labelValue": val,
	})
	return nil
}

// DiagnosticError creates an error using the diagnostic property value.
func DiagnosticError(ctx context.Context, info types.LicenseManagerLicenseInfo) error {
	tflog.Debug(ctx, "Searching for 'diagnostic' property in license info")
	for _, property := range info.Properties {
		tflog.Trace(ctx, "Checking license property", map[string]interface{}{"propertyKey": property.Key})
		if property.Key == "diagnostic" {
			diagnosticValue, ok := property.Value.(string)
			if !ok {
				err := fmt.Errorf("diagnostic property value is not a string (type: %T)", property.Value)
				tflog.Error(ctx, "Invalid type for diagnostic property value", map[string]interface{}{
					"valueType": fmt.Sprintf("%T", property.Value),
					"error":     err.Error(),
				})
				return errors.New("failed to process diagnostic property due to unexpected type")
			}
			tflog.Debug(ctx, "Found 'diagnostic' property, creating error from its value", map[string]interface{}{"diagnosticValue": diagnosticValue})
			return errors.New(diagnosticValue)
		}
	}

	tflog.Debug(ctx, "'diagnostic' property not found in license info")
	return nil
}

// KeyValuesToMap converts a slice of KeyValue objects into a map with string keys and interface{} values.
func KeyValuesToMap(ctx context.Context, keyValues []types.KeyValue) map[string]interface{} {
	mapLen := len(keyValues)
	tflog.Debug(ctx, "Converting KeyValue slice to map", map[string]interface{}{"sliceLength": mapLen})

	resultMap := make(map[string]interface{}, mapLen)
	for _, kv := range keyValues {
		tflog.Trace(ctx, "Adding key-value pair to map", map[string]interface{}{"key": kv.Key})
		resultMap[kv.Key] = kv.Value
	}

	tflog.Debug(ctx, "Successfully converted KeyValue slice to map", map[string]interface{}{"mapLength": len(resultMap)})
	return resultMap
}
