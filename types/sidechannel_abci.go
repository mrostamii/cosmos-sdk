package types

import (
	abci "github.com/tendermint/tendermint/abci/types"
)

//
// side channel related types
//

// BeginSideBlocker runs code before the side transactions in a block
type BeginSideBlocker func(ctx Context, req abci.RequestBeginSideBlock) abci.ResponseBeginSideBlock

// DeliverSideTxHandler runs during each side trasaction in a block
type DeliverSideTxHandler func(ctx Context, tx Tx, req abci.RequestDeliverSideTx) abci.ResponseDeliverSideTx

// PostDeliverTxHandler runs after deliver tx
type PostDeliverTxHandler func(ctx Context, tx Tx, result Result)
