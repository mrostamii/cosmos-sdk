package baseapp

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

//
// Side channel options
//

// SetBeginSideBlocker sets begin side blocker
func (app *BaseApp) SetBeginSideBlocker(beginSideBlocker sdk.BeginSideBlocker) {
	if app.sealed {
		panic("SetBeginSideBlocker() on sealed BaseApp")
	}
	app.beginSideBlocker = beginSideBlocker
}

// SetDeliverSideTxHandler sets deliver side-tx handler
func (app *BaseApp) SetDeliverSideTxHandler(deliverSideTxHandler sdk.DeliverSideTxHandler) {
	if app.sealed {
		panic("SetDeliverSideTxHandler() on sealed BaseApp")
	}
	app.deliverSideTxHandler = deliverSideTxHandler
}

// SetPostDeliverTxHandler sets post deliver tx handler
func (app *BaseApp) SetPostDeliverTxHandler(postDeliverTxHandler sdk.PostDeliverTxHandler) {
	if app.sealed {
		panic("SetPostDeliverTxHandler() on sealed BaseApp")
	}
	app.postDeliverTxHandler = postDeliverTxHandler
}

//
// Side channel ABCI methods
//

// BeginSideBlock implements the ABCI application interface.
func (app *BaseApp) BeginSideBlock(req abci.RequestBeginSideBlock) (res abci.ResponseBeginSideBlock) {
	if app.beginSideBlocker != nil {
		res = app.beginSideBlocker(app.deliverState.ctx, req)
	}

	return
}

// DeliverSideTx implements the ABCI application interface.
func (app *BaseApp) DeliverSideTx(req abci.RequestDeliverSideTx) (res abci.ResponseDeliverSideTx) {
	tx, err := app.txDecoder(req.Tx)
	if err != nil {
		return sdkerrors.ResponseDeliverSideTx(err, 0, 0, app.trace)
	}

	res = app.runSideTx(req.Tx, tx, req)

	return
}

// runSideTx processes a side transaction. App can make an external call here.
func (app *BaseApp) runSideTx(txBytes []byte, tx sdk.Tx, req abci.RequestDeliverSideTx) (res abci.ResponseDeliverSideTx) {
	defer func() {
		if r := recover(); r != nil {
			res = abci.ResponseDeliverSideTx{
				Result:    tmproto.SideTxResultType_SKIP, // skip proposal
				Code:      sdkerrors.ErrUnknownRequest.ABCICode(),
				Codespace: sdkerrors.ErrUnknownRequest.Codespace(),
			}
		}
	}()

	var msgs = tx.GetMsgs()
	if err := validateBasicTxMsgs(msgs); err != nil {
		res = sdkerrors.ResponseDeliverSideTx(err, 0, 0, app.trace)
		return
	}

	if app.deliverSideTxHandler != nil {
		// get deliver-tx context
		ctx := app.getContextForTx(runTxModeDeliver, txBytes)

		res = app.deliverSideTxHandler(ctx, tx, req)
	} else {
		res = abci.ResponseDeliverSideTx{
			Result: tmproto.SideTxResultType_SKIP, // skip proposal
		}
	}

	return
}
