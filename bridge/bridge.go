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

const (
	// Locations
	LOC_HAND    = 0x2
	LOC_DECK    = 0x1
	LOC_GRAVE   = 0x8
	LOC_REMOVED = 0x10
	LOC_MZONE   = 0x4
	LOC_SZONE   = 0x20

	// Positions
	POS_FACEUP   = 0x5
	POS_FACEDOWN = 0xa

	// Duel status codes (returned by ProcessDuel)
	DUEL_STATUS_END      = 0
	DUEL_STATUS_AWAITING = 1
	DUEL_STATUS_CONTINUE = 2
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

func AddCard(duel unsafe.Pointer, code uint32, team uint8, loc uint32, seq uint32, pos uint32) {
	var info C.OCG_NewCardInfo
	info.team = C.uint8_t(team)
	info.duelist = 0
	info.code = C.uint32_t(code)
	info.con = C.uint8_t(team)
	info.loc = C.uint32_t(loc)
	info.seq = C.uint32_t(seq)
	info.pos = C.uint32_t(pos)
	C.OCG_DuelNewCard(C.OCG_Duel(duel), &info)
}

func StartDuel(duel unsafe.Pointer) {
	C.OCG_StartDuel(C.OCG_Duel(duel))
}

func ProcessDuel(duel unsafe.Pointer) int {
	return int(C.OCG_DuelProcess(C.OCG_Duel(duel)))
}

func GetMessage(duel unsafe.Pointer) []byte {
	var length C.uint32_t
	ptr := C.OCG_DuelGetMessage(C.OCG_Duel(duel), &length)
	if ptr == nil || length == 0 {
		return nil
	}
	return C.GoBytes(ptr, C.int(length))
}

// MessageType represents the type of message from the engine
type MessageType byte

const (
	MSG_SELECT_IDLECMD  MessageType = 0x81 // 129 - select idle command
	MSG_SELECT_CHAIN    MessageType = 0x5D // 93  - select chain
	MSG_SELECT_CARD     MessageType = 0x60 // 96  - select card
	MSG_SELECT_YESNO    MessageType = 0x63 // 99  - yes/no prompt
	MSG_SELECT_EFFECTYN MessageType = 0x64 // 100 - activate effect yes/no
	MSG_SELECT_OPTION   MessageType = 0x62 // 98  - select option
	MSG_SELECT_PLACE    MessageType = 0x84 // 132 - select field zone
	MSG_SELECT_POSITION MessageType = 0x61 // 97  - select position
)

type IdleCommand struct {
	CardCode uint32
	Location uint32
	Sequence uint32
}

type IdleCommandMessage struct {
	Player         uint8
	Summons        []IdleCommand
	SpSummons      []IdleCommand
	Repositions    []IdleCommand
	MZoneSetCards  []IdleCommand
	SZoneSetCards  []IdleCommand
	Activations    []IdleCommand
	CanBattlePhase bool
	CanEndPhase    bool
}

func ParseIdleCmd(msg []byte) IdleCommandMessage {
	result := IdleCommandMessage{}

	fmt.Printf("  raw bytes 0-10: %v\n", msg[:10])

	result.Player = msg[1] // player is always byte 1
	pos := 4               // skip: msg_type(1) + player(1) + padding(2) = 4

	fmt.Printf("  [debug] player=%d pos=%d next4bytes=%v\n", result.Player, pos, msg[pos:pos+4])

	readU64 := func() uint64 {
		if pos+8 > len(msg) {
			return 0
		}
		v := uint64(msg[pos]) | uint64(msg[pos+1])<<8 | uint64(msg[pos+2])<<16 |
			uint64(msg[pos+3])<<24 | uint64(msg[pos+4])<<32 | uint64(msg[pos+5])<<40 |
			uint64(msg[pos+6])<<48 | uint64(msg[pos+7])<<56
		pos += 8
		return v
	}

	readU32 := func() uint32 {
		if pos+4 > len(msg) {
			return 0
		}
		v := uint32(msg[pos]) | uint32(msg[pos+1])<<8 |
			uint32(msg[pos+2])<<16 | uint32(msg[pos+3])<<24
		pos += 4
		return v
	}

	readU8 := func() uint8 {
		if pos >= len(msg) {
			return 0
		}
		v := msg[pos]
		pos++
		return v
	}

	// summons: code=u32, ctrl=u8, loc=u8, seq=u32
	count := readU32()
	fmt.Printf("  [debug] summons count=%d\n", count)
	for i := 0; i < int(count); i++ {
		code := readU32()
		_ = readU8() // ctrl
		loc := readU8()
		seq := readU32()
		result.Summons = append(result.Summons, IdleCommand{CardCode: code, Location: uint32(loc), Sequence: seq})
	}

	// spsummons: code=u32, ctrl=u8, loc=u8, seq=u32
	count = readU32()
	for i := 0; i < int(count); i++ {
		code := readU32()
		_ = readU8()
		loc := readU8()
		seq := readU32()
		result.SpSummons = append(result.SpSummons, IdleCommand{CardCode: code, Location: uint32(loc), Sequence: seq})
	}

	// repositions: code=u32, ctrl=u8, loc=u8, seq=u8
	count = readU32()
	for i := 0; i < int(count); i++ {
		code := readU32()
		_ = readU8()
		loc := readU8()
		seq := readU8()
		result.Repositions = append(result.Repositions, IdleCommand{CardCode: code, Location: uint32(loc), Sequence: uint32(seq)})
	}

	// mset: code=u32, ctrl=u8, loc=u8, seq=u32
	count = readU32()
	for i := 0; i < int(count); i++ {
		code := readU32()
		_ = readU8()
		loc := readU8()
		seq := readU32()
		result.MZoneSetCards = append(result.MZoneSetCards, IdleCommand{CardCode: code, Location: uint32(loc), Sequence: seq})
	}

	// sset: code=u32, ctrl=u8, loc=u8, seq=u32
	count = readU32()
	for i := 0; i < int(count); i++ {
		code := readU32()
		_ = readU8()
		loc := readU8()
		seq := readU32()
		result.SZoneSetCards = append(result.SZoneSetCards, IdleCommand{CardCode: code, Location: uint32(loc), Sequence: seq})
	}

	// activations: code=u32, ctrl=u8, loc=u8, seq=u32, desc=u64, mode=u8
	count = readU32()
	for i := 0; i < int(count); i++ {
		code := readU32()
		_ = readU8()
		loc := readU8()
		seq := readU32()
		_ = readU64() // description
		_ = readU8()  // client_mode
		result.Activations = append(result.Activations, IdleCommand{CardCode: code, Location: uint32(loc), Sequence: seq})
	}

	result.CanBattlePhase = readU8() == 1
	result.CanEndPhase = readU8() == 1
	_ = readU8() // can_shuffle

	return result
}
