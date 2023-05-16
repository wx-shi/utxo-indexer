package model

import "github.com/btcsuite/btcd/btcjson"

type UTXO struct {
	Hash    string  `json:"hash"`
	Address string  `json:"address"`
	Index   int     `json:"index"`
	Value   float64 `json:"value"`
	Spend   bool    `json:"-"`
}

type UseUTXO struct {
	btcjson.Vin
	Use struct {
		Hash  string
		Index int
	}
}
type UseInfo struct {
	Hash  string
	Index int
}

// BlockUTXO 块下的utxo 包含已使用的 已经新产生的
type BlockUTXO struct {
	Height int64     `json:"height"`
	Vins   []UseUTXO `json:"vins"`
	Vouts  []UTXO    `json:"vouts"`
}

type UTXORequest struct {
	Address string `json:"address"`
}

type UTXOReply struct {
	Balance string `json:"balance"`
	Utxos   []UTXO `json:"utxos"`
}
