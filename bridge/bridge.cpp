/*
 * C++ implementation of the YGOpen bridge between ygopro-core (ocgapi)
 * and Go (cgo).
 *
 * This file is intentionally conservative: it wires up duel creation,
 * lifecycle, and a skeleton for message encoding and answer application.
 * The actual use of YGOpen's edo9300_ocgcore_encode/Decode is stubbed
 * with TODOs so we can get a compiling, linkable bridge first.
 */

#include "bridge.h"
#include "include/ocgapi.h"

#include <cstdint>
#include <cstdlib>
#include <memory>
#include <vector>

// YGOpen includes
#include <google/protobuf/arena.h>
#include "ygopen/codec/edo9300_ocgcore_encode.hpp"
#include "duel_msg.pb.h"
#include "duel_answer.pb.h"

namespace
{

    using YGOpen::Proto::Duel::Answer;
    using YGOpen::Proto::Duel::Msg;

    extern "C"
    {
        void cardReaderStub(void *payload, uint32_t code, OCG_CardData *data);
        void cardReaderDoneStub(void *payload, OCG_CardData *data);
        int scriptReaderStub(void *payload, OCG_Duel duel, const char *name);
        void logHandlerStub(void *payload, const char *str, int type);
    }

    struct DuelContext : YGOpen::Codec::IEncodeContext
    {
        OCG_Duel duel{nullptr};

        google::protobuf::Arena arena{};
        std::vector<std::string> encoded_msgs{}; // serialized Msg buffers
        std::size_t next_msg_index{0};

        // IEncodeContext implementation – currently minimal/stubbed where possible.

        [[nodiscard]] auto pile_size(Con, Loc) const noexcept -> std::size_t override
        {
            // TODO: optionally use OCG_DuelQueryCount to provide accurate sizes.
            return 0U;
        }

        [[nodiscard]] auto get_match_win_reason() const noexcept -> uint32_t override
        {
            // TODO: track match win reason if needed.
            return 0U;
        }

        [[nodiscard]] auto has_xyz_mat(Place const &) const noexcept -> bool override
        {
            // TODO: implement if you need full xyz material tracking.
            return false;
        }

        [[nodiscard]] auto get_xyz_left(Place const &) const noexcept -> Place override
        {
            // TODO: implement if you need full xyz material tracking.
            return {};
        }

        auto match_win_reason(uint32_t) noexcept -> void override
        {
            // TODO: store match win reason if desired.
        }

        auto xyz_mat_defer(Place const &) noexcept -> void override
        {
            // TODO: implement deferred xyz material handling if needed.
        }

        auto take_deferred_xyz_mat() noexcept -> std::vector<Place> override
        {
            return {};
        }

        auto xyz_left(Place const &, Place const &) noexcept -> void override
        {
            // TODO: implement xyz material movement if needed.
        }
    };

    // Simple helper to cast opaque handle back to context.
    inline DuelContext *ctx_from_handle(YGO_DuelHandle handle)
    {
        return static_cast<DuelContext *>(handle);
    }

} // namespace

int ygo_duel_create(YGO_DuelHandle *out_handle, const YGO_DuelOptions *opts)
{
    if (!out_handle || !opts)
    {
        return -1;
    }

    auto *ctx = new (std::nothrow) DuelContext();
    if (!ctx)
    {
        return -2;
    }

    OCG_DuelOptions options{};
    options.seed[0] = opts->seed[0];
    options.seed[1] = opts->seed[1];
    options.seed[2] = opts->seed[2];
    options.seed[3] = opts->seed[3];

    options.team1.startingLP = opts->starting_lp;
    options.team1.startingDrawCount = opts->starting_draw_count;
    options.team1.drawCountPerTurn = opts->draw_count_per_turn;

    options.team2.startingLP = opts->starting_lp;
    options.team2.startingDrawCount = opts->starting_draw_count;
    options.team2.drawCountPerTurn = opts->draw_count_per_turn;

    options.cardReader = cardReaderStub;
    options.cardReaderDone = cardReaderDoneStub;
    options.scriptReader = scriptReaderStub;
    options.logHandler = logHandlerStub;

    auto status = OCG_CreateDuel(&ctx->duel, &options);
    if (status != OCG_DUEL_CREATION_SUCCESS)
    {
        delete ctx;
        return -3;
    }

    *out_handle = static_cast<YGO_DuelHandle>(ctx);
    return 0;
}

void ygo_duel_destroy(YGO_DuelHandle handle)
{
    auto *ctx = ctx_from_handle(handle);
    if (!ctx)
    {
        return;
    }
    if (ctx->duel)
    {
        OCG_DestroyDuel(ctx->duel);
        ctx->duel = nullptr;
    }
    delete ctx;
}

void ygo_duel_add_card(YGO_DuelHandle handle,
                       uint8_t team,
                       uint8_t duelist,
                       uint32_t code,
                       uint8_t con,
                       uint32_t loc,
                       uint32_t seq,
                       uint32_t pos)
{
    auto *ctx = ctx_from_handle(handle);
    if (!ctx || !ctx->duel)
    {
        return;
    }

    OCG_NewCardInfo info{};
    info.team = team;
    info.duelist = duelist;
    info.code = code;
    info.con = con;
    info.loc = loc;
    info.seq = seq;
    info.pos = pos;

    OCG_DuelNewCard(ctx->duel, &info);
}

void ygo_duel_start(YGO_DuelHandle handle)
{
    auto *ctx = ctx_from_handle(handle);
    if (!ctx || !ctx->duel)
    {
        return;
    }
    OCG_StartDuel(ctx->duel);
}

int ygo_duel_step(YGO_DuelHandle handle)
{
    auto *ctx = ctx_from_handle(handle);
    if (!ctx || !ctx->duel)
    {
        return -1;
    }

    ctx->encoded_msgs.clear();
    ctx->next_msg_index = 0;

    auto status = OCG_DuelProcess(ctx->duel);

    if (status == OCG_DUEL_STATUS_AWAITING)
    {
        // Fetch raw core messages and encode them to YGOpen protos.
        uint32_t length = 0;
        void *raw = OCG_DuelGetMessage(ctx->duel, &length);
        if (raw && length > 0)
        {
            auto *data = static_cast<uint8_t *>(raw);

            // TODO: Use YGOpen::Codec::Edo9300::OCGCore::encode_one here
            // in a loop to turn raw core messages into Msg objects and
            // serialize them into ctx->encoded_msgs.
            //
            // For now, we do not attempt to parse the raw bytes; the
            // bridge will report AWAITING but ygo_duel_next_msg will
            // return 0 (no messages). This is enough to get the bridge
            // compiling so we can iterate on the encoding next.
            (void)data;
        }
    }

    return static_cast<int>(status);
}

int ygo_duel_next_msg(YGO_DuelHandle handle, YGO_Buffer *out_buf)
{
    if (!out_buf)
    {
        return -1;
    }

    auto *ctx = ctx_from_handle(handle);
    if (!ctx)
    {
        return -1;
    }

    if (ctx->next_msg_index >= ctx->encoded_msgs.size())
    {
        out_buf->data = nullptr;
        out_buf->len = 0;
        return 0;
    }

    auto &s = ctx->encoded_msgs[ctx->next_msg_index++];
    out_buf->data = reinterpret_cast<const uint8_t *>(s.data());
    out_buf->len = static_cast<uint32_t>(s.size());
    return 1;
}

int ygo_duel_apply_answer(YGO_DuelHandle handle,
                          const uint8_t *data,
                          uint32_t len)
{
    auto *ctx = ctx_from_handle(handle);
    if (!ctx || !ctx->duel || !data || len == 0U)
    {
        return -1;
    }

    // Decode the Answer protobuf.
    Answer answer;
    if (!answer.ParseFromArray(data, static_cast<int>(len)))
    {
        return -2;
    }

    // TODO: Inspect answer.t() oneof and translate to the raw response
    // format expected by ygopro-core. For now this is a stub that does
    // nothing but report success so that higher layers can be wired.
    //
    // Example for SelectIdle (IDLECMD) will eventually:
    //  - Read answer.select_idle().card_action().action()/index()
    //  - Pack them into a 4-byte value as in existing Go code
    //  - Call OCG_DuelSetResponse(ctx->duel, buffer, 4)

    (void)answer;
    return 0;
}