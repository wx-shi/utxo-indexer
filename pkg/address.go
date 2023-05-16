package pkg

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

// GetAddressByScriptPubKeyResult 获取地址
func GetAddressByScriptPubKeyResult(sp btcjson.ScriptPubKeyResult) (string, error) {
	// 从十六进制字符串解码脚本
	script, err := hex.DecodeString(sp.Hex)
	if err != nil {
		return "", err
	}

	// 解析脚本
	_, addresses, _, err := txscript.ExtractPkScriptAddrs(script, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}

	if len(addresses) > 0 {
		return addresses[0].EncodeAddress(), nil
	}

	return "", fmt.Errorf("unable to extract address from scriptPubKeyResult")

}
