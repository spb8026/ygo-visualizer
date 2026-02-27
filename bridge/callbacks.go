package bridge

/*
#include "bridge.h"
*/
import "C"
import "unsafe"

//export goCardReader
func goCardReader(code C.uint32_t, data *C.OCG_CardData) {
	globalCardMu.RLock()
	db := globalCardDB
	globalCardMu.RUnlock()
	if db == nil {
		return
	}
	card, err := db.GetCard(uint32(code))
	if err != nil || card == nil {
		return
	}
	data.code = C.uint32_t(card.Code)
	data.alias = C.uint32_t(card.Alias)
	data._type = C.uint32_t(card.Type)
	data.level = C.uint32_t(card.Level)
	data.attribute = C.uint32_t(card.Attribute)
	data.race = C.uint64_t(card.Race)
	data.attack = C.int32_t(card.Attack)
	data.defense = C.int32_t(card.Defense)
	data.lscale = C.uint32_t(card.Lscale)
	data.rscale = C.uint32_t(card.Rscale)
	data.link_marker = C.uint32_t(card.LinkMarker)
}

//export goScriptReader
func goScriptReader(duel unsafe.Pointer, name *C.char) C.int {
	// TODO: implement script loading
	return 0
}

//export goLogHandler
func goLogHandler(str *C.char, logType C.int) {
	// TODO: implement logging
}
