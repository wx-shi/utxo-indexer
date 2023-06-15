package test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

func TestAddress(t *testing.T) {
	address, err := GetAddressByScriptPubKeyResult(btcjson.ScriptPubKeyResult{
		Asm:     "040a464653204c756b652d4a72206c656176652074686520626c6f636b636861696e20616c6f6e65210a4f682c20616e6420676f642069736e2774207265616c0a OP_CHECKSIG",
		Hex:     "41040a464653204c756b652d4a72206c656176652074686520626c6f636b636861696e20616c6f6e65210a4f682c20616e6420676f642069736e2774207265616c0aac",
		ReqSigs: 0,
		Type:    "pubkey",
	})

	t.Log(address, err)

}
func GetAddressByScriptPubKeyResult(sp btcjson.ScriptPubKeyResult) (string, error) {
	switch sp.Type {
	case "pubkey":
		data, err := hex.DecodeString(sp.Hex[2 : len(sp.Hex)-2])
		if err != nil {
			return "", err
		}
		addr, err := btcutil.NewAddressPubKey(data, &chaincfg.MainNetParams)
		if err != nil {
			return "", err
		}
		return addr.EncodeAddress(), err
	}
	return "", nil
}

func TestXx(t *testing.T) {
	fmt.Println(1 << 20)
}
