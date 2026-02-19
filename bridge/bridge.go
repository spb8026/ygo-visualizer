package bridge

/*
#cgo LDFLAGS: -L${SRCDIR} -locgcore
#include "ocgapi.h"
#include <stdlib.h>
#include <string.h>

// cardReader: called by engine to get card stats by code.
// For now we return a blank card so the engine doesn't crash.
void cardReaderStub(void* payload, uint32_t code, OCG_CardData* data) {
    memset(data, 0, sizeof(OCG_CardData));
    data->code = code;
}

// cardReaderDone: called after cardReader so we can free memory.
// Nothing to free in our stub.
void cardReaderDoneStub(void* payload, OCG_CardData* data) {}

// scriptReader: called by engine to load a Lua script by name.
// Returning 0 means "script not found" — engine will skip it.
int scriptReaderStub(void* payload, OCG_Duel duel, const char* name) {
    return 0;
}

// logHandler: called by engine to emit log messages.
void logHandlerStub(void* payload, const char* str, int type) {
    // We'll wire this to Go's logger later
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func GetVersion() (int, int) {
	var major, minor C.int
	C.OCG_GetVersion(&major, &minor)
	return int(major), int(minor)
}

func CreateDuel() (unsafe.Pointer, error) {
	var options C.OCG_DuelOptions

	// Zero out the whole struct first
	options.seed[0] = 12345
	options.seed[1] = 0
	options.seed[2] = 0
	options.seed[3] = 0

	// Set up players with standard LP and draw counts
	options.team1.startingLP = 8000
	options.team1.startingDrawCount = 5
	options.team1.drawCountPerTurn = 1

	options.team2.startingLP = 8000
	options.team2.startingDrawCount = 5
	options.team2.drawCountPerTurn = 1

	// Wire up the callbacks
	options.cardReader = C.OCG_DataReader(C.cardReaderStub)
	options.cardReaderDone = C.OCG_DataReaderDone(C.cardReaderDoneStub)
	options.scriptReader = C.OCG_ScriptReader(C.scriptReaderStub)
	options.logHandler = C.OCG_LogHandler(C.logHandlerStub)

	var duel C.OCG_Duel
	status := C.OCG_CreateDuel(&duel, &options)

	if status != C.OCG_DUEL_CREATION_SUCCESS {
		return nil, fmt.Errorf("OCG_CreateDuel failed with status: %d", int(status))
	}

	return unsafe.Pointer(duel), nil
}
