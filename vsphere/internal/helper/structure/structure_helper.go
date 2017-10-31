package structure

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi/vim25/types"
)

// SliceInterfacesToStrings converts an interface slice to a string slice. The
// function does not attempt to do any sanity checking and will panic if one of
// the items in the slice is not a string.
func SliceInterfacesToStrings(s []interface{}) []string {
	var d []string
	for _, v := range s {
		d = append(d, v.(string))
	}
	return d
}

// SliceStringsToInterfaces converts a string slice to an interface slice.
func SliceStringsToInterfaces(s []string) []interface{} {
	var d []interface{}
	for _, v := range s {
		d = append(d, v)
	}
	return d
}

// MergeSchema merges the map[string]*schema.Schema from src into dst. Safety
// against conflicts is enforced by panicing.
func MergeSchema(dst, src map[string]*schema.Schema) {
	for k, v := range src {
		if _, ok := dst[k]; ok {
			panic(fmt.Errorf("conflicting schema key: %s", k))
		}
		dst[k] = v
	}
}

// BoolPtr makes a *bool out of the value passed in through v.
//
// vSphere uses nil values in bools to omit values in the SOAP XML request, and
// helps denote inheritance in certain cases.
func BoolPtr(v bool) *bool {
	return &v
}

// GetBoolPtr reads a ResourceData and returns an appropriate *bool for the
// state of the definition. nil is returned if it does not exist.
func GetBoolPtr(d *schema.ResourceData, key string) *bool {
	v, e := d.GetOkExists(key)
	if e {
		return BoolPtr(v.(bool))
	}
	return nil
}

// GetBool reads a ResourceData and returns a *bool. This differs from
// GetBoolPtr in that a nil value is never returned.
func GetBool(d *schema.ResourceData, key string) *bool {
	return BoolPtr(d.Get(key).(bool))
}

// SetBoolPtr sets a ResourceData field depending on if a *bool exists or not.
// The field is not set if it's nil.
func SetBoolPtr(d *schema.ResourceData, key string, val *bool) error {
	if val == nil {
		return nil
	}
	err := d.Set(key, val)
	return err
}

// Int64Ptr makes an *int64 out of the value passed in through v.
func Int64Ptr(v int64) *int64 {
	return &v
}

// Int32Ptr makes an *int32 out of the value passed in through v.
func Int32Ptr(v int32) *int32 {
	return &v
}

// GetInt64Ptr reads a ResourceData and returns an appropriate *int64 for the
// state of the definition. nil is returned if it does not exist.
func GetInt64Ptr(d *schema.ResourceData, key string) *int64 {
	v, e := d.GetOkExists(key)
	if e {
		return Int64Ptr(int64(v.(int)))
	}
	return nil
}

// SetInt64Ptr sets a ResourceData field depending on if an *int64 exists or
// not.  The field is not set if it's nil.
func SetInt64Ptr(d *schema.ResourceData, key string, val *int64) error {
	if val == nil {
		return nil
	}
	err := d.Set(key, val)
	return err
}

// ByteToMB returns n/1000000. The input must be an integer that can be divisible
// by 1000000.
func ByteToMB(n interface{}) interface{} {
	switch v := n.(type) {
	case int:
		return v / 1000000
	case int32:
		return v / 1000000
	case int64:
		return v / 1000000
	}
	panic(fmt.Errorf("non-integer type %T for value", n))
}

// ByteToGB returns n/1000000000. The input must be an integer that can be
// divisible by 1000000000.
//
// Remember that int32 overflows at 2GB, so any values higher than that will
// produce an inaccurate result.
func ByteToGB(n interface{}) interface{} {
	switch v := n.(type) {
	case int:
		return v / 1000000000
	case int32:
		return v / 1000000000
	case int64:
		return v / 1000000000
	}
	panic(fmt.Errorf("non-integer type %T for value", n))
}

// GbToByte returns n*1000000000.
//
// The output is returned as int64 - if another type is needed, it needs to be
// cast. Remember that int32 overflows at 2GB and uint32 will overflow at 4GB.
func GbToByte(n interface{}) int64 {
	switch v := n.(type) {
	case int:
		return int64(v * 1000000000)
	case int32:
		return int64(v * 1000000000)
	case int64:
		return v * 1000000000
	}
	panic(fmt.Errorf("non-integer type %T for value", n))
}

// BoolPolicy converts a bool into a VMware BoolPolicy value.
func BoolPolicy(b bool) *types.BoolPolicy {
	bp := &types.BoolPolicy{
		Value: BoolPtr(b),
	}
	return bp
}

// GetBoolPolicy reads a ResourceData and returns an appropriate BoolPolicy for
// the state of the definition. nil is returned if it does not exist.
func GetBoolPolicy(d *schema.ResourceData, key string) *types.BoolPolicy {
	v, e := d.GetOkExists(key)
	if e {
		return BoolPolicy(v.(bool))
	}
	return nil
}

// SetBoolPolicy sets a ResourceData field depending on if a BoolPolicy exists
// or not. The field is not set if it's nil.
func SetBoolPolicy(d *schema.ResourceData, key string, val *types.BoolPolicy) error {
	if val == nil {
		return nil
	}
	err := d.Set(key, val.Value)
	return err
}

// GetBoolPolicyReverse acts like GetBoolPolicy, but the value is inverted.
func GetBoolPolicyReverse(d *schema.ResourceData, key string) *types.BoolPolicy {
	v, e := d.GetOkExists(key)
	if e {
		return BoolPolicy(!v.(bool))
	}
	return nil
}

// SetBoolPolicyReverse acts like SetBoolPolicy, but the value is inverted.
func SetBoolPolicyReverse(d *schema.ResourceData, key string, val *types.BoolPolicy) error {
	if val == nil {
		return nil
	}
	err := d.Set(key, !*val.Value)
	return err
}

// StringPolicy converts a string into a VMware StringPolicy value.
func StringPolicy(s string) *types.StringPolicy {
	sp := &types.StringPolicy{
		Value: s,
	}
	return sp
}

// GetStringPolicy reads a ResourceData and returns an appropriate StringPolicy
// for the state of the definition. nil is returned if it does not exist.
func GetStringPolicy(d *schema.ResourceData, key string) *types.StringPolicy {
	v, e := d.GetOkExists(key)
	if e {
		return StringPolicy(v.(string))
	}
	return nil
}

// SetStringPolicy sets a ResourceData field depending on if a StringPolicy
// exists or not. The field is not set if it's nil.
func SetStringPolicy(d *schema.ResourceData, key string, val *types.StringPolicy) error {
	if val == nil {
		return nil
	}
	err := d.Set(key, val.Value)
	return err
}

// LongPolicy converts a supported number into a VMware LongPolicy value. This
// will panic if there is no implicit conversion of the value into an int64.
func LongPolicy(n interface{}) *types.LongPolicy {
	lp := &types.LongPolicy{}
	switch v := n.(type) {
	case int:
		lp.Value = int64(v)
	case int8:
		lp.Value = int64(v)
	case int16:
		lp.Value = int64(v)
	case int32:
		lp.Value = int64(v)
	case uint:
		lp.Value = int64(v)
	case uint8:
		lp.Value = int64(v)
	case uint16:
		lp.Value = int64(v)
	case uint32:
		lp.Value = int64(v)
	case int64:
		lp.Value = v
	default:
		panic(fmt.Errorf("non-convertible type %T for value", n))
	}
	return lp
}

// GetLongPolicy reads a ResourceData and returns an appropriate LongPolicy
// for the state of the definition. nil is returned if it does not exist.
func GetLongPolicy(d *schema.ResourceData, key string) *types.LongPolicy {
	v, e := d.GetOkExists(key)
	if e {
		return LongPolicy(v)
	}
	return nil
}

// SetLongPolicy sets a ResourceData field depending on if a LongPolicy
// exists or not. The field is not set if it's nil.
func SetLongPolicy(d *schema.ResourceData, key string, val *types.LongPolicy) error {
	if val == nil {
		return nil
	}
	err := d.Set(key, val.Value)
	return err
}

// AllFieldsEmpty checks to see if all fields in a given struct are zero
// values. It does not recurse, so finer-grained checking should be done for
// deep accuracy when necessary. It also does not dereference pointers, except
// if the value itself is a pointer and is not nil.
func AllFieldsEmpty(v interface{}) bool {
	if v == nil {
		return true
	}

	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Struct && (t.Kind() == reflect.Ptr && t.Elem().Kind() != reflect.Struct) {
		if reflect.Zero(t).Interface() != reflect.ValueOf(v).Interface() {
			return false
		}
		return true
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		var fv reflect.Value
		if reflect.ValueOf(v).Kind() == reflect.Ptr {
			fv = reflect.ValueOf(v).Elem().Field(i)
		} else {
			fv = reflect.ValueOf(v).Elem().Field(i)
		}

		ft := t.Field(i).Type
		fz := reflect.Zero(ft)
		switch ft.Kind() {
		case reflect.Map, reflect.Slice:
			if fv.Len() > 0 {
				return false
			}
		default:
			if fz.Interface() != fv.Interface() {
				return false
			}
		}
	}

	return true
}

// DeRef returns the value pointed to by the interface if the interface is a
// pointer and is not nil, otherwise returns nil, or the direct value if it's
// not a pointer.
func DeRef(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	k := reflect.TypeOf(v).Kind()
	if k != reflect.Ptr {
		return v
	}
	if reflect.ValueOf(v) == reflect.Zero(reflect.TypeOf(v)) {
		// All zero-value pointers are nil
		return nil
	}
	return reflect.ValueOf(v).Elem().Interface()
}
