package bridge

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -lygopen -locgcore -lprotobuf -lstdc++ -L"C:/msys64/ucrt64/lib" -labsl_log_internal_message -labsl_log_internal_check_op -labsl_log_internal_conditions -labsl_log_internal_format -labsl_log_internal_nullguard -labsl_log_internal_proto -labsl_log_internal_globals -labsl_log_internal_log_sink_set -labsl_log_globals -labsl_log_sink -labsl_log_entry -labsl_raw_logging_internal -labsl_strings -labsl_strings_internal -labsl_string_view -labsl_base -labsl_spinlock_wait -labsl_throw_delegate -labsl_int128 -labsl_synchronization -labsl_time -labsl_time_zone -labsl_civil_time -labsl_status -labsl_strerror
#cgo CFLAGS: -I${SRCDIR} -I${SRCDIR}/include
#cgo CXXFLAGS: -I${SRCDIR} -I${SRCDIR}/include
#include "bridge.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/spb8026/ygo-visualizer/carddb"
	answerpb "github.com/spb8026/ygo-visualizer/ygopenpb" // from duel_answer.proto
	duelpb "github.com/spb8026/ygo-visualizer/ygopenpb"   // from duel_msg.proto
	"google.golang.org/protobuf/proto"
)

/*
   ----------------------------------------------------------------------------
   Card DB glue used by callbacks.go
   ----------------------------------------------------------------------------
*/

var (
	globalCardDB *carddb.DB
	globalCardMu sync.RWMutex
)

func SetCardDB(db *carddb.DB) {
	globalCardMu.Lock()
	defer globalCardMu.Unlock()
	globalCardDB = db
}

/*
   ----------------------------------------------------------------------------
   High-level Duel wrapper over the C bridge
   ----------------------------------------------------------------------------
*/

type Duel struct {
	h C.YGO_DuelHandle
}

type DuelOptions struct {
	Seed              [4]uint64
	StartingLP        uint32
	StartingDrawCount uint32
	DrawCountPerTurn  uint32
}

func NewDuel(opts DuelOptions) (*Duel, error) {
	var cOptions C.YGO_DuelOptions
	for i := 0; i < 4; i++ {
		cOptions.seed[i] = C.uint64_t(opts.Seed[i])
	}
	cOptions.starting_lp = C.uint32_t(opts.StartingLP)
	cOptions.starting_draw_count = C.uint32_t(opts.StartingDrawCount)
	cOptions.draw_count_per_turn = C.uint32_t(opts.DrawCountPerTurn)

	var h C.YGO_DuelHandle
	rc := C.ygo_duel_create(&h, &cOptions)
	if rc != 0 {
		return nil, fmt.Errorf("ygo_duel_create failed: %d", int(rc))
	}
	return &Duel{h: h}, nil
}

func (d *Duel) Close() {
	if d.h != nil {
		C.ygo_duel_destroy(d.h)
		d.h = nil
	}
}

/*
   Location / position / status constants (re-exported for convenience)
*/

const (
	LOC_HAND    = 0x2
	LOC_DECK    = 0x1
	LOC_GRAVE   = 0x8
	LOC_REMOVED = 0x10
	LOC_MZONE   = 0x4
	LOC_SZONE   = 0x20

	POS_FACEUP   = 0x5
	POS_FACEDOWN = 0xa
)

type DuelStatus int

const (
	DuelStatusEnd      DuelStatus = 0
	DuelStatusAwaiting DuelStatus = 1
	DuelStatusContinue DuelStatus = 2
)

/*
   Wrapper helpers for setup and stepping
*/

func (d *Duel) AddCard(team, duelist uint8, code uint32, con uint8, loc, seq, pos uint32) {
	C.ygo_duel_add_card(
		d.h,
		C.uint8_t(team),
		C.uint8_t(duelist),
		C.uint32_t(code),
		C.uint8_t(con),
		C.uint32_t(loc),
		C.uint32_t(seq),
		C.uint32_t(pos),
	)
}

func (d *Duel) Start() {
	C.ygo_duel_start(d.h)
}

// Step advances the duel one tick and returns:
//   - DuelStatus (END/AWAITING/CONTINUE)
//   - all encoded YGOpen Duel.Msg protobufs produced in this step (as raw bytes)
func (d *Duel) Step() (DuelStatus, [][]byte, error) {
	rc := C.ygo_duel_step(d.h)
	if rc < 0 {
		return 0, nil, fmt.Errorf("ygo_duel_step failed: %d", int(rc))
	}
	status := DuelStatus(rc)

	var msgs [][]byte
	for {
		var buf C.YGO_Buffer
		has := C.ygo_duel_next_msg(d.h, &buf)
		if has < 0 {
			return status, nil, fmt.Errorf("ygo_duel_next_msg failed")
		}
		if has == 0 {
			break // no more messages
		}

		b := C.GoBytes(unsafe.Pointer(buf.data), C.int(buf.len))
		msgs = append(msgs, b)
	}

	return status, msgs, nil
}

/*
   Optional: keep MessageType for compatibility with existing code
   (this is based on the old raw core protocol; you may later remove it)
*/

type MessageType byte

const (
	MSG_RETRY                MessageType = 1
	MSG_HINT                 MessageType = 2
	MSG_WAITING              MessageType = 3
	MSG_START                MessageType = 4
	MSG_WIN                  MessageType = 5
	MSG_UPDATE_DATA          MessageType = 6
	MSG_UPDATE_CARD          MessageType = 7
	MSG_REQUEST_DECK         MessageType = 8
	MSG_SELECT_BATTLECMD     MessageType = 10
	MSG_SELECT_IDLECMD       MessageType = 11
	MSG_SELECT_IDLECMD_2     MessageType = 0xE5
	MSG_SELECT_IDLECMD_3     MessageType = 0x9B
	MSG_SELECT_EFFECTYN      MessageType = 12
	MSG_SELECT_YESNO         MessageType = 13
	MSG_SELECT_OPTION        MessageType = 14
	MSG_SELECT_CARD          MessageType = 15
	MSG_SELECT_CHAIN         MessageType = 16
	MSG_SELECT_PLACE         MessageType = 18
	MSG_SELECT_POSITION      MessageType = 19
	MSG_SELECT_TRIBUTE       MessageType = 20
	MSG_SORT_CHAIN           MessageType = 21
	MSG_SELECT_COUNTER       MessageType = 22
	MSG_SELECT_SUM           MessageType = 23
	MSG_SELECT_DISFIELD      MessageType = 24
	MSG_SORT_CARD            MessageType = 25
	MSG_SELECT_UNSELECT_CARD MessageType = 26
	MSG_CONFIRM_DECKTOP      MessageType = 30
	MSG_CONFIRM_CARDS        MessageType = 31
	MSG_SHUFFLE_DECK         MessageType = 32
	MSG_SHUFFLE_HAND         MessageType = 33
	MSG_REFRESH_DECK         MessageType = 34
	MSG_SWAP_GRAVE_DECK      MessageType = 35
	MSG_SHUFFLE_SET_CARD     MessageType = 36
	MSG_REVERSE_DECK         MessageType = 37
	MSG_DECK_TOP             MessageType = 38
	MSG_SHUFFLE_EXTRA        MessageType = 39
	MSG_NEW_TURN             MessageType = 40
	MSG_NEW_PHASE            MessageType = 41
	MSG_CONFIRM_EXTRATOP     MessageType = 42
	MSG_MOVE                 MessageType = 50
	MSG_POS_CHANGE           MessageType = 53
	MSG_SET                  MessageType = 54
	MSG_SWAP                 MessageType = 55
	MSG_FIELD_DISABLED       MessageType = 56
	MSG_SUMMONING            MessageType = 60
	MSG_SUMMONED             MessageType = 61
	MSG_SPSUMMONING          MessageType = 62
	MSG_SPSUMMONED           MessageType = 63
	MSG_FLIPSUMMONING        MessageType = 64
	MSG_FLIPSUMMONED         MessageType = 65
	MSG_CHAINING             MessageType = 70
	MSG_CHAINED              MessageType = 71
	MSG_CHAIN_SOLVING        MessageType = 72
	MSG_CHAIN_SOLVED         MessageType = 73
	MSG_CHAIN_END            MessageType = 74
	MSG_CHAIN_NEGATED        MessageType = 75
	MSG_CHAIN_DISABLED       MessageType = 76
	MSG_CARD_SELECTED        MessageType = 80
	MSG_RANDOM_SELECTED      MessageType = 81
	MSG_BECOME_TARGET        MessageType = 83
	MSG_DRAW                 MessageType = 90
	MSG_DAMAGE               MessageType = 91
	MSG_RECOVER              MessageType = 92
	MSG_EQUIP                MessageType = 93
	MSG_LPUPDATE             MessageType = 94
	MSG_UNEQUIP              MessageType = 95
	MSG_CARD_TARGET          MessageType = 96
	MSG_CANCEL_TARGET        MessageType = 97
	MSG_PAY_LPCOST           MessageType = 100
	MSG_ADD_COUNTER          MessageType = 101
	MSG_REMOVE_COUNTER       MessageType = 102
	MSG_ATTACK               MessageType = 110
	MSG_BATTLE               MessageType = 111
	MSG_ATTACK_DISABLED      MessageType = 112
	MSG_DAMAGE_STEP_START    MessageType = 113
	MSG_DAMAGE_STEP_END      MessageType = 114
	MSG_MISSED_EFFECT        MessageType = 120
	MSG_BE_CHAIN_TARGET      MessageType = 121
	MSG_CREATE_RELATION      MessageType = 122
	MSG_RELEASE_RELATION     MessageType = 123
	MSG_TOSS_COIN            MessageType = 130
	MSG_TOSS_DICE            MessageType = 131
	MSG_ROCK_PAPER_SCISSORS  MessageType = 132
	MSG_HAND_RES             MessageType = 133
	MSG_ANNOUNCE_RACE        MessageType = 140
	MSG_ANNOUNCE_ATTRIB      MessageType = 141
	MSG_ANNOUNCE_CARD        MessageType = 142
	MSG_ANNOUNCE_NUMBER      MessageType = 143
	MSG_CARD_HINT            MessageType = 160
	MSG_TAG_SWAP             MessageType = 161
	MSG_RELOAD_FIELD         MessageType = 162
	MSG_AI_NAME              MessageType = 163
	MSG_SHOW_HINT            MessageType = 164
	MSG_PLAYER_HINT          MessageType = 165
	MSG_MATCH_KILL           MessageType = 170
	MSG_CUSTOM_MSG           MessageType = 180
	MSG_REMOVE_CARDS         MessageType = 190
)

func (m MessageType) String() string {
	switch m {
	case MSG_RETRY:
		return "MSG_RETRY"
	case MSG_SELECT_IDLECMD:
		return "MSG_SELECT_IDLECMD"
	case MSG_SELECT_BATTLECMD:
		return "MSG_SELECT_BATTLECMD"
	case MSG_SELECT_CARD:
		return "MSG_SELECT_CARD"
	case MSG_SELECT_PLACE:
		return "MSG_SELECT_PLACE"
	case MSG_SELECT_POSITION:
		return "MSG_SELECT_POSITION"
	case MSG_SELECT_EFFECTYN:
		return "MSG_SELECT_EFFECTYN"
	case MSG_SELECT_YESNO:
		return "MSG_SELECT_YESNO"
	case MSG_SELECT_OPTION:
		return "MSG_SELECT_OPTION"
	case MSG_WIN:
		return "MSG_WIN"
	case MSG_DRAW:
		return "MSG_DRAW"
	case MSG_DAMAGE:
		return "MSG_DAMAGE"
	case MSG_LPUPDATE:
		return "MSG_LPUPDATE"
	case MSG_SUMMONING:
		return "MSG_SUMMONING"
	case MSG_SUMMONED:
		return "MSG_SUMMONED"
	case MSG_ATTACK:
		return "MSG_ATTACK"
	case MSG_BATTLE:
		return "MSG_BATTLE"
	case MSG_MOVE:
		return "MSG_MOVE"
	case MSG_NEW_TURN:
		return "MSG_NEW_TURN"
	case MSG_NEW_PHASE:
		return "MSG_NEW_PHASE"
	default:
		return fmt.Sprintf("MSG_UNKNOWN(0x%02X)", byte(m))
	}
}

/*
   Example helper: decode all Duel.Msg values for a single step.
   You can call this from your CLI / graph builder.
*/

func HandleStep(d *Duel) error {
	status, rawMsgs, err := d.Step()
	if err != nil {
		return err
	}

	_ = status // you can inspect this if needed

	for _, b := range rawMsgs {
		var m duelpb.Msg
		if err := proto.Unmarshal(b, &m); err != nil {
			return err
		}

		if req := m.GetRequest(); req != nil {
			if sel := req.GetSelectIdle(); sel != nil {
				// TODO: plug into your visualizer / graph builder.
				_ = sel
			}
		}
	}

	return nil
}

/*
   Example writer: build and send an Answer.SelectIdle from Go
*/

func (d *Duel) SendIdleCardAction(action answerpb.Answer_SelectIdle_Action, index uint32) error {
	ans := &answerpb.Answer{
		T: &answerpb.Answer_SelectIdle_{
			SelectIdle: &answerpb.Answer_SelectIdle{
				T: &answerpb.Answer_SelectIdle_CardAction_{
					CardAction: &answerpb.Answer_SelectIdle_CardAction{
						Action: action,
						Index:  index,
					},
				},
			},
		},
	}

	b, err := proto.Marshal(ans)
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}

	rc := C.ygo_duel_apply_answer(
		d.h,
		(*C.uint8_t)(unsafe.Pointer(&b[0])),
		C.uint32_t(len(b)),
	)
	if rc != 0 {
		return fmt.Errorf("ygo_duel_apply_answer failed: %d", int(rc))
	}

	return nil
}
