package indexer

import (
	"github.com/avast/retry-go"
	"github.com/btcsuite/btcd/btcjson"
)

func (i *Indexer) getBlockTx(height int64) (*btcjson.GetBlockVerboseTxResult, error) {
	f := func() (*btcjson.GetBlockVerboseTxResult, error) {
		hash, err := i.rpc.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
		res, err := i.rpc.GetBlockVerboseTx(hash)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	resp, err := f()
	if err != nil {
		_ = retry.Do(func() error {
			resp, err = f()
			return err
		}, retry.Attempts(3))
	}
	return resp, err
}
