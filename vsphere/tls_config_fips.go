// +build : fips

package vsphere

import (
	"crypto/tls"
)

func getTlsConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12}
}
