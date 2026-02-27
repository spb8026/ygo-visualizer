/*
 * YGOpen bridge between ygopro-core (ocgapi) and Go (cgo).
 *
 * This C header exposes a small, C-compatible API that:
 *  - Wraps an internal duel context (ygopro-core duel + YGOpen encoder state).
 *  - Steps the duel and buffers any encoded YGOpen::Proto::Duel::Msg messages.
 *  - Accepts serialized YGOpen::Proto::Duel::Answer messages as input.
 *
 * The implementation lives in bridge.cpp and is compiled as C++,
 * but this header is safe to include from C and cgo.
 */

#ifndef BRIDGE_H
#define BRIDGE_H

#include <stdint.h>
#include "include/ocgapi.h"

#ifdef __cplusplus
extern "C" {
#endif

/* ----------------------------------------------------------------------------
 * CALLBACK STUBS (Bridges C++ calls to Go functions)
 * ----------------------------------------------------------------------------
 */

// These are defined in callbacks.go using //export
extern void goCardReader(uint32_t code, OCG_CardData* data);
extern int goScriptReader(void* duel, char* name);
extern void goLogHandler(char* str, int type);

// Use static inline to ensure the body is visible to both bridge.cpp and Go
static inline void cardReaderStub(void* payload, uint32_t code, OCG_CardData* data) {
    goCardReader(code, data);
}

static inline void cardReaderDoneStub(void* payload, OCG_CardData* data) {
    // No-op
}

static inline int scriptReaderStub(void* payload, OCG_Duel duel, const char* name) {
    return goScriptReader(duel, (char*)name);
}

static inline void logHandlerStub(void* payload, const char* str, int type) {
    goLogHandler((char*)str, type);
}

/* ----------------------------------------------------------------------------
 * BRIDGE API
 * ----------------------------------------------------------------------------
 */

typedef void* YGO_DuelHandle;

typedef struct YGO_Buffer {
    const uint8_t* data;
    uint32_t len;
} YGO_Buffer;

enum {
    YGO_DUEL_STATUS_END      = 0,
    YGO_DUEL_STATUS_AWAITING = 1,
    YGO_DUEL_STATUS_CONTINUE = 2,
};

typedef struct YGO_DuelOptions {
    uint64_t seed[4];
    uint32_t starting_lp;
    uint32_t starting_draw_count;
    uint32_t draw_count_per_turn;
} YGO_DuelOptions;

int ygo_duel_create(YGO_DuelHandle* out_handle, const YGO_DuelOptions* opts);
void ygo_duel_destroy(YGO_DuelHandle handle);
void ygo_duel_add_card(YGO_DuelHandle handle, uint8_t team, uint8_t duelist, uint32_t code, uint8_t con, uint32_t loc, uint32_t seq, uint32_t pos);
void ygo_duel_start(YGO_DuelHandle handle);
int ygo_duel_step(YGO_DuelHandle handle);
int ygo_duel_next_msg(YGO_DuelHandle handle, YGO_Buffer* out_buf);
int ygo_duel_apply_answer(YGO_DuelHandle handle, const uint8_t* data, uint32_t len);

#ifdef __cplusplus
}
#endif

#endif /* BRIDGE_H */
 