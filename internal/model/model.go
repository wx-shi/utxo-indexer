package model

// BlockUTXO 块下的utxo 包含已使用的 已经新产生的
type BlockUTXO struct {
	Height int64 `json:"height"`
	Vins   []In  `json:"vins"`
	Vouts  []Out `json:"vouts"`
}

// 花费
type In struct {
	UKey  string
	TxID  string
	Index int
	Spend *Spend // txid:index
}

type Spend struct {
	TxID  string
	Index int
}

// 新入
type Out struct {
	UKey    string
	TxID    string
	Index   int
	Address string  `json:"address"`
	Value   float64 `json:"value"`
}

type UTXORequest struct {
	Address  string `json:"address"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type UTXO struct {
	TxID  string `json:"tx_id"`
	Index int    `json:"index"`
	Value string `json:"value"`
}

type UTXOReply struct {
	Balance   string  `json:"balance"`
	Page      int     `json:"page"`
	PageSize  int     `json:"page_size"`
	TotalSize int     `json:"total_size"`
	Utxos     []*UTXO `json:"utxos"`
}

type HeightReply struct {
	StoreHeight int64 `json:"store_height"`
	NodeHeight  int64 `json:"node_height"`
}
