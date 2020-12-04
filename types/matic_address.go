package types

import (
	"errors"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ethCommon "github.com/maticnetwork/bor/common"
)

// String implements the Stringer interface.
func (aa AccAddress) String() string {
	if aa.Empty() {
		return ""
	}

	return ethCommon.ToHex(aa)
}

// Bech32ifyPubKey returns a Bech32 encoded string containing the appropriate
// prefix based on the key type provided for a given PublicKey.
// TODO: Remove Bech32ifyPubKey and all usages (cosmos/cosmos-sdk/issues/#7357)
func Bech32ifyPubKey(pkt Bech32PubKeyType, pubkey cryptotypes.PubKey) (string, error) {
	return ethCommon.ToHex(pubkey.Bytes()), nil
}

// GetFromBech32 decodes a bytestring from a Bech32 encoded string.
func GetFromBech32(bech32str, prefix string) ([]byte, error) {
	return ethCommon.FromHex(bech32str), nil
}

func addressBytesFromHexString(address string) ([]byte, error) {
	if len(address) == 0 {
		return nil, errors.New("decoding address failed: must provide a valid address")
	}

	return ethCommon.FromHex(address), nil
}
