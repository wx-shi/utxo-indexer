package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/wx-shi/utxo-indexer/internal/config"
	"github.com/wx-shi/utxo-indexer/internal/db"
	"github.com/wx-shi/utxo-indexer/internal/model"
	"github.com/wx-shi/utxo-indexer/pkg"
	"go.uber.org/zap"
)

type Indexer struct {
	ctx                 context.Context
	logger              *zap.Logger
	rpc                 *rpcclient.Client
	db                  *db.DB
	conf                *config.IndexerConfig
	scanHeight          int64
	storeHeight         int64
	blockChan           chan model.BlockUTXO
	isHistoryScanFinish bool
	Finish              chan struct{}
}

func NewIndexer(ctx context.Context, conf *config.IndexerConfig,
	logger *zap.Logger, rpc *rpcclient.Client, db *db.DB) *Indexer {
	return &Indexer{
		ctx:    ctx,
		conf:   conf,
		logger: logger,
		rpc:    rpc,
		db:     db,
		Finish: make(chan struct{}),
	}
}

func (i *Indexer) Sync() {
	i.init()
	// i.fixBalance()
	go i.scan()
	go i.store()
	return
}

func (i *Indexer) init() {
	// init
	height, err := i.db.GetStoreHeight()
	if err != nil {
		i.logger.Fatal("GetStoreHeight", zap.Error(err))
	}
	i.storeHeight = height
	i.scanHeight = height + 1
	i.blockChan = make(chan model.BlockUTXO, i.conf.BlockChanBuf)
}

func (i *Indexer) scan() {
	for {
		select {
		case <-i.ctx.Done():
			return
		default:

			//获取当前最新高度
			nheight, err := i.rpc.GetBlockCount()
			if err != nil {
				i.logger.Error("GetBlockCount", zap.Error(err))
				continue
			}

			if i.scanHeight > nheight {
				continue
			}

			i.isHistoryScanFinish = false

			if err := i.scanByHeightRange(i.scanHeight, nheight); err != nil {
				i.logger.Error("scanByHeightRange", zap.Error(err))
				continue
			}

		}
	}
}

// scanByHeightRange 扫描 通过高度范围
func (idx *Indexer) scanByHeightRange(startHeight int64, endHeight int64) error {
	for i := startHeight; i <= endHeight; i++ {
		if i == endHeight {
			idx.isHistoryScanFinish = true
		}
		if err := idx.scanTxByBlock(i); err != nil {
			idx.logger.Error("scanTxByBlock", zap.Int64("height", i), zap.Error(err))
			return err
		}
		idx.scanHeight = i + 1
	}
	return nil
}

// scanTxByBlock 扫描指定高度
func (idx *Indexer) scanTxByBlock(height int64) error {

	startTime := time.Now()
	btxs, err := idx.getBlockTx(height)
	if err != nil {
		idx.logger.Error("getBlockTx", zap.Int64("height", height), zap.Error(err))
		return err
	}

	vins := make([]model.In, 0, 10000)
	vouts := make([]model.Out, 0, 10000)
	for _, tx := range btxs.Tx {
		for i, vin := range tx.Vin {
			//判断是否为coinbase
			if len(vin.Coinbase) > 0 || len(vin.Txid) == 0 {
				continue
			}
			vins = append(vins, model.In{
				UKey:  fmt.Sprintf("u:%s:%d", vin.Txid, vin.Vout),
				TxID:  vin.Txid,
				Index: int(vin.Vout),
				Spend: &model.Spend{
					TxID:  tx.Txid,
					Index: i,
				},
			})
		}
		for i, vout := range tx.Vout {
			switch vout.ScriptPubKey.Type {
			case txscript.NonStandardTy.String(),
				txscript.NullDataTy.String():
				continue
			default:
				address, err := pkg.GetAddressByScriptPubKeyResult(vout.ScriptPubKey)
				if err != nil || len(address) == 0 {
					// todo debug
					idx.logger.Debug("GetAddressByScriptPubKeyResult",
						zap.Any("vout", vout),
						zap.String("txid", tx.Txid),
						zap.Int("index", i),
						zap.Error(err))
					continue
				}
				vouts = append(vouts, model.Out{
					UKey:    fmt.Sprintf("u:%s:%d", tx.Txid, i),
					TxID:    tx.Txid,
					Index:   i,
					Address: address,
					Value:   vout.Value,
				})
			}
		}
	}
	idx.blockChan <- model.BlockUTXO{
		Height: height,
		Vins:   vins,
		Vouts:  vouts,
	}

	idx.logger.Debug("Scan::Info", zap.Int64("height", height), zap.Int("tx_len", len(btxs.Tx)), zap.Duration("ttl", time.Since(startTime)))
	return nil
}

func (i *Indexer) store() {
	vins := make([]model.In, 0, 1000000)
	vouts := make([]model.Out, 0, 1000000)
	var lastHeight int64
	for {
		select {
		case <-i.ctx.Done():
			close(i.Finish) //确保存储完成后退出
			return
		case hUtxos := <-i.blockChan:
			lastHeight = hUtxos.Height
			vins = append(vins, hUtxos.Vins...)
			vouts = append(vouts, hUtxos.Vouts...)
			if i.isHistoryScanFinish {
				//直接存储
				if err := i.db.Store(vins, vouts, lastHeight); err == nil {
					vins = make([]model.In, 0, 1000000)
					vouts = make([]model.Out, 0, 1000000)
				}
				continue
			}
		}
		//如果10w个utxo进行存储
		if len(vins)+len(vouts) >= int(i.conf.BatchSize) {
			if err := i.db.Store(vins, vouts, lastHeight); err == nil {
				vins = make([]model.In, 0, 1000000)
				vouts = make([]model.Out, 0, 1000000)
			}
		}
	}
}
