package virtualdisk

import (
	"reflect"
	"testing"

	"github.com/vmware/govmomi/object"
)

func TestDatastorePathFromString(t *testing.T) {
	cases := []struct {
		name     string
		subject  string
		expected *object.DatastorePath
		success  bool
	}{
		{
			name:     "standard",
			subject:  "[datastore1] foo/bar.vmdk",
			expected: &object.DatastorePath{Datastore: "datastore1", Path: "foo/bar.vmdk"},
			success:  true,
		},
		{
			name:     "no datastore",
			subject:  "foo/bar.vmdk",
			expected: &object.DatastorePath{},
			success:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, success := DatastorePathFromString(tc.subject)
			if !reflect.DeepEqual(tc.expected, actual) {
				t.Fatalf("expected %+v, got %+v", tc.expected, actual)
			}
			if tc.success != success {
				t.Fatalf("expected success to be %t, got %t", tc.success, success)
			}
		})
	}
}

func TestIsVmdkDatastorePath(t *testing.T) {
	cases := []struct {
		name     string
		subject  string
		expected bool
	}{
		{
			name:     "correct",
			subject:  "[datastore1] foo/bar.vmdk",
			expected: true,
		},
		{
			name:     "incorrect - no datastore",
			subject:  "foo/bar.vmdk",
			expected: false,
		},
		{
			name:     "incorrect - does not end in .vmdk",
			subject:  "[datastore1] foo/bar",
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := IsVmdkDatastorePath(tc.subject)
			if tc.expected != actual {
				t.Fatalf("expected %t, got %t", tc.expected, actual)
			}
		})
	}
}

func TestDstDataStorePathFromLocalSrc(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		dst      string
		expected string
	}{
		{
			name:     "local path, no dir",
			src:      "[datastore1] foo/bar.vmdk",
			dst:      "baz.vmdk",
			expected: "[datastore1] foo/baz.vmdk",
		},
		{
			name:     "same datastore, full dir",
			src:      "[datastore1] foo/bar.vmdk",
			dst:      "bar/baz.vmdk",
			expected: "[datastore1] bar/baz.vmdk",
		},
		{
			name:     "different datastore, no dir (should override datastore)",
			src:      "[datastore1] foo/bar.vmdk",
			dst:      "[datastore2] baz.vmdk",
			expected: "[datastore1] foo/baz.vmdk",
		},
		{
			name:     "bad path, no dir (should still go thru)",
			src:      "[datastore1] foo/bar.vmdk",
			dst:      "baz",
			expected: "[datastore1] foo/baz",
		},
		{
			name:     "bad path, full dir (should still go thru)",
			src:      "[datastore1] foo/bar.vmdk",
			dst:      "baz/qux",
			expected: "[datastore1] baz/qux",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := dstDataStorePathFromLocalSrc(tc.src, tc.dst)
			if tc.expected != actual {
				t.Fatalf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}
