package registration

import (
	"encoding/hex"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/enigmampc/EnigmaBlockchain/x/registration/internal/types"
	ra "github.com/enigmampc/EnigmaBlockchain/x/registration/remote_attestation"
)

const (
	AttributeSigner        = "signer"
	AttributeEncryptedSeed = "encrypted_seed"
	AttributeNodeID        = "node_id"
)

// NewHandler returns a handler for "bank" type messages.
func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {

		case MsgRaAuthenticate:
			return handleRaAuthenticate(ctx, k, &msg)
		case *MsgRaAuthenticate:
			return handleRaAuthenticate(ctx, k, msg)

		default:
			errMsg := fmt.Sprintf("unrecognized wasm message type: %T", msg)
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	}
}

// filterMessageEvents returns the same events with all of type == EventTypeMessage removed.
// this is so only our top-level message event comes through
func filterMessageEvents(manager *sdk.EventManager) sdk.Events {
	events := manager.Events()
	res := make([]sdk.Event, 0, len(events)+1)
	for _, e := range events {
		if e.Type != sdk.EventTypeMessage {
			res = append(res, e)
		}
	}
	return res
}

func handleRaAuthenticate(ctx sdk.Context, k Keeper, msg *types.RaAuthenticate) (*sdk.Result, error) {
	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	pubkey, err := ra.VerifyRaCert(msg.Certificate)
	if err != nil {
		return nil, err
	}

	// fmt.Println("RaAuth", hex.EncodeToString(msg.PubKey))
	encSeed, err := k.RegisterNode(ctx, msg.Certificate)
	if err != nil {
		return nil, err
	}

	events := filterMessageEvents(ctx.EventManager())
	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
		sdk.NewAttribute(AttributeSigner, msg.Sender.String()),
		sdk.NewAttribute(AttributeEncryptedSeed, fmt.Sprintf("0x%02x", encSeed)),
		sdk.NewAttribute(AttributeNodeID, fmt.Sprintf("0x%s", hex.EncodeToString(pubkey))),
	)

	pkIO := k.GetMasterPublicKey(ctx)

	return &sdk.Result{
		Data:   []byte(fmt.Sprintf("S: %02x | PK: %s", encSeed, hex.EncodeToString(*pkIO))),
		Events: append(events, ourEvent),
	}, nil
}