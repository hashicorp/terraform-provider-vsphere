package provider

import (
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
)

// DefaultAPITimeout is a default timeout value that is passed to functions
// requiring contexts, and other various waiters.
const DefaultAPITimeout = time.Minute * 5

// Log is just a simple wrapper for the current logging pattern with the format
// string pumped through spew.Sprintf. This is so that we get better dumping of
// complex structures versus just stopping at pointers. The caller still needs
// to set the level.
func Log(format string, v ...interface{}) {
	log.Println(spew.Sprintf("[DEBUG] "+format, v...))
}
