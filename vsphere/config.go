package vsphere

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/debug"
	"github.com/vmware/vic/pkg/vsphere/tags"
	"golang.org/x/net/context"
)

// VSphereClient is the client connection manager for the vSphere provider. It
// holds the connections to the various API endpoints we need to interface
// with, such as the VMODL API through govmomi, and the REST SDK through
// alternate libraries.
type VSphereClient struct {
	// The VIM/govmomi client.
	vimClient *govmomi.Client

	// The specialized tags client SDK imported from vmware/vic.
	tagsClient *tags.RestClient
}

// TagsClient returns the embedded REST client used for tags, after determining
// if the connection is eligible:
//
// * The connection information in vimClient is valid vCenter connection
// * The provider has a connection to the CIS REST client. This is true if
// tagsClient != nil.
//
// This function should be used whenever possible to return the client from the
// provider meta variable for use, to determine if it can be used at all.
//
// The nil value that is returned on an unsupported connection can be
// considered stable behavior for read purposes on resources that need to be
// able to read tags if they are present. You can use the snippet below in a
// Read call to determine if tags are supported on this connection, and if they
// are, read them from the object and save them in the resource:
//
//   if tagsClient, _ := meta.(*VSphereClient).TagsClient(); tagsClient != nil {
//     if err := readTagsForResource(tagsClient, obj, d); err != nil {
//       return err
//     }
//   }
func (c *VSphereClient) TagsClient() (*tags.RestClient, error) {
	if err := viapi.ValidateVirtualCenter(c.vimClient); err != nil {
		return nil, err
	}
	if c.tagsClient == nil {
		return nil, fmt.Errorf("tags require %s or higher", tagsMinVersion)
	}
	return c.tagsClient, nil
}

// Config holds the provider configuration, and delivers a populated
// VSphereClient based off the contained settings.
type Config struct {
	User          string
	Password      string
	VSphereServer string
	InsecureFlag  bool
	Debug         bool
	DebugPath     string
	DebugPathRun  string
}

// Client returns a new client for accessing VMWare vSphere.
func (c *Config) Client() (*VSphereClient, error) {
	client := new(VSphereClient)

	u, err := url.Parse("https://" + c.VSphereServer + "/sdk")
	if err != nil {
		return nil, fmt.Errorf("Error parse url: %s", err)
	}

	u.User = url.UserPassword(c.User, c.Password)

	err = c.EnableDebug()
	if err != nil {
		return nil, fmt.Errorf("Error setting up client debug: %s", err)
	}

	// Set up the VIM/govmomi client connection.
	vctx, vcancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer vcancel()
	client.vimClient, err = govmomi.NewClient(vctx, u, c.InsecureFlag)
	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	log.Printf("[INFO] VMWare vSphere Client configured for URL: %s", c.VSphereServer)

	// Skip the rest of this function if we are not setting up the tags client. This is if
	if !isEligibleTagEndpoint(client.vimClient) {
		log.Printf("[WARN] Connected endpoint does not support tags (%s)", viapi.ParseVersionFromClient(client.vimClient))
		return client, nil
	}

	// Otherwise, connect to the CIS REST API for tagging.
	log.Printf("[INFO] Logging in to CIS REST API endpoint on %s", c.VSphereServer)
	client.tagsClient = tags.NewClient(u, c.InsecureFlag, "")
	tctx, tcancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer tcancel()
	if err := client.tagsClient.Login(tctx); err != nil {
		return nil, fmt.Errorf("Error connecting to CIS REST endpoint: %s", err)
	}
	// Done
	log.Println("[INFO] CIS REST login successful")

	return client, nil
}

// EnableDebug turns on govmomi API operation logging, if appropriate settings
// are set on the provider.
func (c *Config) EnableDebug() error {
	if !c.Debug {
		return nil
	}

	// Base path for storing debug logs.
	r := c.DebugPath
	if r == "" {
		r = filepath.Join(os.Getenv("HOME"), ".govmomi")
	}
	r = filepath.Join(r, "debug")

	// Path for this particular run.
	run := c.DebugPathRun
	if run == "" {
		now := time.Now().Format("2006-01-02T15-04-05.999999999")
		r = filepath.Join(r, now)
	} else {
		// reuse the same path
		r = filepath.Join(r, run)
		_ = os.RemoveAll(r)
	}

	err := os.MkdirAll(r, 0700)
	if err != nil {
		log.Printf("[ERROR] Client debug setup failed: %v", err)
		return err
	}

	p := debug.FileProvider{
		Path: r,
	}

	debug.SetProvider(&p)
	return nil
}
