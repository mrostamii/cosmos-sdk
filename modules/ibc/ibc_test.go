package ibc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/light-client/certifiers"
	"github.com/tendermint/tmlibs/log"

	"github.com/tendermint/basecoin"
	"github.com/tendermint/basecoin/errors"
	"github.com/tendermint/basecoin/stack"
	"github.com/tendermint/basecoin/state"
)

type checkErr func(error) bool

func noErr(err error) bool {
	return err == nil
}

func genEmptySeed(keys certifiers.ValKeys, chain string, h int,
	appHash []byte, count int) certifiers.Seed {

	vals := keys.ToValidators(10, 0)
	cp := keys.GenCheckpoint(chain, h, nil, vals, appHash, 0, count)
	return certifiers.Seed{cp, vals}
}

// this tests registration without registrar permissions
func TestIBCRegister(t *testing.T) {
	assert := assert.New(t)

	// the validators we use to make seeds
	keys := certifiers.GenValKeys(5)
	keys2 := certifiers.GenValKeys(7)
	appHash := []byte{0, 4, 7, 23}
	appHash2 := []byte{12, 34, 56, 78}

	// badSeed doesn't validate
	badSeed := genEmptySeed(keys2, "chain-2", 123, appHash, len(keys2))
	badSeed.Header.AppHash = appHash2

	cases := []struct {
		seed    certifiers.Seed
		checker checkErr
	}{
		{
			genEmptySeed(keys, "chain-1", 100, appHash, len(keys)),
			noErr,
		},
		{
			genEmptySeed(keys, "chain-1", 200, appHash, len(keys)),
			IsAlreadyRegisteredErr,
		},
		{
			badSeed,
			IsInvalidCommitErr,
		},
		{
			genEmptySeed(keys2, "chain-2", 123, appHash2, 5),
			noErr,
		},
	}

	ctx := stack.MockContext("hub", 50)
	store := state.NewMemKVStore()
	app := stack.New().Dispatch(stack.WrapHandler(NewHandler()))

	for i, tc := range cases {
		tx := RegisterChainTx{tc.seed}.Wrap()
		_, err := app.DeliverTx(ctx, store, tx)
		assert.True(tc.checker(err), "%d: %+v", i, err)
	}
}

// this tests permission controls on ibc registration
func TestIBCRegisterPermissions(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// the validators we use to make seeds
	keys := certifiers.GenValKeys(4)
	appHash := []byte{0x17, 0x21, 0x5, 0x1e}

	foobar := basecoin.Actor{App: "foo", Address: []byte("bar")}
	baz := basecoin.Actor{App: "baz", Address: []byte("bar")}
	foobaz := basecoin.Actor{App: "foo", Address: []byte("baz")}

	cases := []struct {
		seed      certifiers.Seed
		registrar basecoin.Actor
		signer    basecoin.Actor
		checker   checkErr
	}{
		// no sig, no registrar
		{
			seed:    genEmptySeed(keys, "chain-1", 100, appHash, len(keys)),
			checker: noErr,
		},
		// sig, no registrar
		{
			seed:    genEmptySeed(keys, "chain-2", 100, appHash, len(keys)),
			signer:  foobaz,
			checker: noErr,
		},
		// registrar, no sig
		{
			seed:      genEmptySeed(keys, "chain-3", 100, appHash, len(keys)),
			registrar: foobar,
			checker:   errors.IsUnauthorizedErr,
		},
		// registrar, wrong sig
		{
			seed:      genEmptySeed(keys, "chain-4", 100, appHash, len(keys)),
			signer:    foobaz,
			registrar: foobar,
			checker:   errors.IsUnauthorizedErr,
		},
		// registrar, wrong sig
		{
			seed:      genEmptySeed(keys, "chain-5", 100, appHash, len(keys)),
			signer:    baz,
			registrar: foobar,
			checker:   errors.IsUnauthorizedErr,
		},
		// registrar, proper sig
		{
			seed:      genEmptySeed(keys, "chain-6", 100, appHash, len(keys)),
			signer:    foobar,
			registrar: foobar,
			checker:   noErr,
		},
	}

	store := state.NewMemKVStore()
	app := stack.New().Dispatch(stack.WrapHandler(NewHandler()))

	for i, tc := range cases {
		// set option specifies the registrar
		msg, err := json.Marshal(tc.registrar)
		require.Nil(err, "%+v", err)
		_, err = app.SetOption(log.NewNopLogger(), store,
			NameIBC, OptionRegistrar, string(msg))
		require.Nil(err, "%+v", err)

		// add permissions to the context
		ctx := stack.MockContext("hub", 50).WithPermissions(tc.signer)
		tx := RegisterChainTx{tc.seed}.Wrap()
		_, err = app.DeliverTx(ctx, store, tx)
		assert.True(tc.checker(err), "%d: %+v", i, err)
	}
}

// this verifies that we can properly update the headers on the chain
func TestIBCUpdate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// this is the root seed, that others are evaluated against
	keys := certifiers.GenValKeys(7)
	appHash := []byte{0, 4, 7, 23}
	start := 100 // initial height
	root := genEmptySeed(keys, "chain-1", 100, appHash, len(keys))

	keys2 := keys.Extend(2)
	keys3 := keys2.Extend(2)

	// create the app and register the root of trust (for chain-1)
	ctx := stack.MockContext("hub", 50)
	store := state.NewMemKVStore()
	app := stack.New().Dispatch(stack.WrapHandler(NewHandler()))
	tx := RegisterChainTx{root}.Wrap()
	_, err := app.DeliverTx(ctx, store, tx)
	require.Nil(err, "%+v", err)

	cases := []struct {
		seed    certifiers.Seed
		checker checkErr
	}{
		// same validator, higher up
		{
			genEmptySeed(keys, "chain-1", start+50, []byte{22}, len(keys)),
			noErr,
		},
		// same validator, between existing (not most recent)
		{
			genEmptySeed(keys, "chain-1", start+5, []byte{15, 43}, len(keys)),
			noErr,
		},
		// same validators, before root of trust
		{
			genEmptySeed(keys, "chain-1", start-8, []byte{11, 77}, len(keys)),
			IsHeaderNotFoundErr,
		},
		// insufficient signatures
		{
			genEmptySeed(keys, "chain-1", start+60, []byte{24}, len(keys)/2),
			IsInvalidCommitErr,
		},
		// unregistered chain
		{
			genEmptySeed(keys, "chain-2", start+60, []byte{24}, len(keys)/2),
			IsNotRegisteredErr,
		},
		// too much change (keys -> keys3)
		{
			genEmptySeed(keys3, "chain-1", start+100, []byte{22}, len(keys3)),
			IsInvalidCommitErr,
		},
		// legit update to validator set (keys -> keys2)
		{
			genEmptySeed(keys2, "chain-1", start+90, []byte{33}, len(keys2)),
			noErr,
		},
		// now impossible jump works (keys -> keys2 -> keys3)
		{
			genEmptySeed(keys3, "chain-1", start+100, []byte{44}, len(keys3)),
			noErr,
		},
	}

	for i, tc := range cases {
		tx := UpdateChainTx{tc.seed}.Wrap()
		_, err := app.DeliverTx(ctx, store, tx)
		assert.True(tc.checker(err), "%d: %+v", i, err)
	}
}

func TestIBCCreatePacket(t *testing.T) {

}

func TestIBCPostPacket(t *testing.T) {

}

func TestIBCSendTx(t *testing.T) {

}
