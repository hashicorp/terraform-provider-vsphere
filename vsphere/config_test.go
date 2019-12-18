package vsphere

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
)

func testAccClientPreCheck(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC to run vsphere_virtual_machine state migration tests (provider connection is required)")
	}
	testAccPreCheck(t)
}

func testAccClientGenerateConfig(t *testing.T) *Config {
	insecure, _ := strconv.ParseBool(os.Getenv("VSPHERE_ALLOW_UNVERIFIED_SSL"))
	debug, _ := strconv.ParseBool(os.Getenv("VSPHERE_CLIENT_DEBUG"))

	return &Config{
		InsecureFlag:  insecure,
		Debug:         debug,
		User:          os.Getenv("VSPHERE_USER"),
		Password:      os.Getenv("VSPHERE_PASSWORD"),
		VSphereServer: os.Getenv("VSPHERE_SERVER"),
		DebugPath:     os.Getenv("VSPHERE_CLIENT_DEBUG_PATH"),
		DebugPathRun:  os.Getenv("VSPHERE_CLIENT_DEBUG_PATH_RUN"),
	}
}

func testAccClientGenerateData(t *testing.T, c *Config) (string, string) {
	_, err := c.Client()
	if err != nil {
		t.Fatalf("error setting up client: %s", err)
	}

	vimSessionFile, err := c.vimSessionFile()
	if err != nil {
		t.Fatalf("error computing VIM session file: %s", err)
	}
	restSessionFile, err := c.restSessionFile()
	if err != nil {
		t.Fatalf("error computing REST session file: %s", err)
	}

	vimData, err := ioutil.ReadFile(vimSessionFile)
	if err != nil {
		t.Fatalf("error reading VIM session file: %s", err)
	}

	restData, err := ioutil.ReadFile(restSessionFile)
	if err != nil {
		t.Fatalf("error reading REST session file: %s", err)
	}

	return string(vimData), string(restData)
}

func testAccClientCheckStatNoExist(t *testing.T, p string) {
	_, err := os.Stat(p)
	switch {
	case err == nil:
		t.Fatalf("expected session file %q to not exist", p)
	case err != nil && os.IsNotExist(err):
		return
	case err != nil:
		t.Fatalf("could not stat path %q: %s", p, err)
	}
}

func TestAccClient_persistence(t *testing.T) {
	testAccClientPreCheck(t)

	vimSessionDir, err := ioutil.TempDir("", "tf-vsphere-test-vimsessiondir")
	if err != nil {
		t.Fatalf("error creating VIM session temp directory: %s", err)
	}
	restSessionDir, err := ioutil.TempDir("", "tf-vsphere-test-restsessiondir")
	if err != nil {
		t.Fatalf("error creating REST session temp directory: %s", err)
	}
	defer func() {
		if err = os.RemoveAll(vimSessionDir); err != nil {
			log.Printf("[DEBUG] Error removing test VIM session directory %q: %s", vimSessionDir, err)
		}
	}()
	defer func() {
		if err = os.RemoveAll(restSessionDir); err != nil {
			log.Printf("[DEBUG] Error removing test REST session directory %q: %s", restSessionDir, err)
		}
	}()

	c := testAccClientGenerateConfig(t)
	c.Persist = true
	c.VimSessionPath = vimSessionDir
	c.RestSessionPath = restSessionDir

	expectedVim, expectedRest := testAccClientGenerateData(t, c)

	// This will create a brand new session under normal circumstances
	actualVim, actualRest := testAccClientGenerateData(t, c)

	if expectedVim != actualVim {
		t.Fatalf("VIM session data mismatch.\n\n\n\nExpected:\n\n %s\n\nActual:\n\n%s\n\n", expectedVim, actualVim)
	}
	if expectedRest != actualRest {
		t.Fatalf("REST session data mismatch.\n\n\n\nExpected:\n\n %s\n\nActual:\n\n%s\n\n", expectedRest, actualRest)
	}
}

func TestAccClient_noPersistence(t *testing.T) {
	testAccClientPreCheck(t)

	vimSessionDir, err := ioutil.TempDir("", "tf-vsphere-test-vimsessiondir")
	if err != nil {
		t.Fatalf("error creating VIM session temp directory: %s", err)
	}
	restSessionDir, err := ioutil.TempDir("", "tf-vsphere-test-restsessiondir")
	if err != nil {
		t.Fatalf("error creating REST session temp directory: %s", err)
	}
	defer func() {
		if err = os.RemoveAll(vimSessionDir); err != nil {
			log.Printf("[DEBUG] Error removing test VIM session directory %q: %s", vimSessionDir, err)
		}
	}()
	defer func() {
		if err = os.RemoveAll(restSessionDir); err != nil {
			log.Printf("[DEBUG] Error removing test REST session directory %q: %s", restSessionDir, err)
		}
	}()

	c := testAccClientGenerateConfig(t)
	// Just to be explicit on intent
	c.Persist = false
	c.VimSessionPath = vimSessionDir
	c.RestSessionPath = restSessionDir

	_, err = c.Client()
	if err != nil {
		t.Fatalf("error setting up client: %s", err)
	}

	vimSessionFile, err := c.vimSessionFile()
	if err != nil {
		t.Fatalf("error computing VIM session file: %s", err)
	}
	restSessionFile, err := c.restSessionFile()
	if err != nil {
		t.Fatalf("error computing REST session file: %s", err)
	}

	testAccClientCheckStatNoExist(t, vimSessionFile)
	testAccClientCheckStatNoExist(t, restSessionFile)
}

func TestNewConfig(t *testing.T) {
	expected := &Config{
		User:            "foo",
		Password:        "bar",
		InsecureFlag:    true,
		VSphereServer:   "vsphere.foo.internal",
		Debug:           true,
		DebugPathRun:    "./foo",
		DebugPath:       "./bar",
		Persist:         true,
		VimSessionPath:  "./baz",
		RestSessionPath: "./qux",
	}

	r := &schema.Resource{Schema: Provider().(*schema.Provider).Schema}
	d := r.Data(nil)
	d.Set("user", expected.User)
	d.Set("password", expected.Password)
	d.Set("allow_unverified_ssl", expected.InsecureFlag)
	d.Set("vsphere_server", expected.VSphereServer)
	d.Set("client_debug", expected.Debug)
	d.Set("client_debug_path_run", expected.DebugPathRun)
	d.Set("client_debug_path", expected.DebugPath)
	d.Set("persist_session", expected.Persist)
	d.Set("vim_session_path", expected.VimSessionPath)
	d.Set("rest_session_path", expected.RestSessionPath)

	actual, err := NewConfig(d)
	if err != nil {
		t.Fatalf("error creating new configuration: %s", err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %#v, got %#v", expected, actual)
	}
}
