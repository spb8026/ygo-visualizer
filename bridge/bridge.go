package bridge

/*
#cgo LDFLAGS: -L${SRCDIR} -locgcore
#include "ocgapi.h"
#include <stdlib.h>
*/
import "C"

func GetVersion() (int, int) {
	var major, minor C.int
	C.OCG_GetVersion(&major, &minor)
	return int(major), int(minor)
}
