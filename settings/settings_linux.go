// NOTE: Clean separation of OS dependent code!

package settings

import (
	"os"
)

func setNyfikenRoot() {
	NyfikenRoot = os.Getenv("HOME") + "/.config/nyfiken"
}
