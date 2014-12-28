package settings

import (
	"os"
)

// NOTE: Nyfiken seems to work fine on Windows even when using "/" for paths.
// Would this method also work for older versions of Windows (e.g. XP)? If not
// it may be worth using filepath.Join(os.Getenv("APPDATA"), "nyfiken") instead,
// and use the same kind of change in other places of the code.

func setNyfikenRoot() {
	NyfikenRoot = os.Getenv("APPDATA") + "/nyfiken"
}
