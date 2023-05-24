package db

import (
	"context"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"strconv"
	"strings"
	"time"

	"github.com/scylladb/go-set/strset"
	"google.golang.org/protobuf/proto"

	"github.com/dgraph-io/badger/v3/options"
	"github.com/shopspring/decimal"
	"github.com/wx-shi/utxo-indexer/internal/config"
	"github.com/wx-shi/utxo-indexer/internal/model"
	"github.com/wx-shi/utxo-indexer/pkg"
	"go.uber.org/zap"
)

const (
	addressBalanceKeyPrefix = "ab:"
	addressUtxoKeyPrefix    = "au:"
	defaultMapCap           = 10000
	StoreHeight             = "store::height"
)

// BadgerDB is a wrapper around the badger.DB instance.
type BadgerDB struct {
	*badger.DB
	logger *zap.Logger
}

// NewBadgerDB creates a new BadgerDB instance.
func NewBadgerDB(config *config.BadgerDBConfig, logger *zap.Logger) (*BadgerDB, error) {
	opts := badger.DefaultOptions(config.Directory).
		WithMemTableSize(256 << 20).    //调整内存表大小
		WithBlockCacheSize(2000 << 20). //调整块缓存大小
		WithLevelSizeMultiplier(20).    // 调整级别大小乘数以减少文件合并
		WithNumMemtables(10).           // 增加内存表数量，以减少磁盘写入
		//WithDetectConflicts(false).      // 如果不需要事务冲突检测，禁用它以提高写入性能
		WithCompression(options.Snappy). //使用Snappy压缩以减少存储空间
		WithLoggingLevel(badger.WARNING)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &BadgerDB{
		DB:     db,
		logger: logger,
	}, nil
}

func (db *BadgerDB) GC(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(20 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := db.RunValueLogGC(0.1); err != nil &&
					err != badger.ErrNoRewrite && err != badger.ErrRejected {
					panic(err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (db *BadgerDB) GetStoreHeight() (int64, error) {
	var height int64
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(StoreHeight))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		height = pkg.BytesToInt64(val)
		return nil
	})
	return height, err
}

func (db *BadgerDB) GetUTXOByAddress(address string, page int, pageSize int) (*model.UTXOReply, error) {

	abKey := addressBalanceKeyPrefix + address
	auKey := addressUtxoKeyPrefix + address

	reply := &model.UTXOReply{
		Page:     page,
		PageSize: pageSize,
	}
	err := db.View(func(txn *badger.Txn) error {
		//获取余额
		bitem, err := txn.Get([]byte(abKey))
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		if err == badger.ErrKeyNotFound {
			var v float64
			reply.Balance = fmt.Sprintf("%.8f", v)
			return nil
		}
		err = bitem.Value(func(val []byte) error {
			pf, err := strconv.ParseFloat(string(val), 64)
			if err != nil {
				return err
			}
			reply.Balance = fmt.Sprintf("%.8f", pf)
			return nil
		})

		//获取utxo列表
		uitem, err := txn.Get([]byte(auKey))
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		if err == badger.ErrKeyNotFound {
			return nil
		}
		us := &StringSet{}
		if err := uitem.Value(func(val []byte) error {
			return proto.Unmarshal(val, us)
		}); err != nil {
			return err
		}

		reply.TotalSize = len(us.Members)
		uarr := pkg.Paginate(us.Members, page, pageSize)
		utxos := make([]*model.UTXO, 0, len(uarr))

		for _, ukey := range uarr {
			keyArr := strings.Split(ukey, ":")
			if len(keyArr) != 3 {
				return fmt.Errorf("invalid key:%s", ukey)
			}
			item, err := txn.Get([]byte(ukey))
			if err != nil {
				return err
			}
			info := &UtxoInfo{}
			if err := item.Value(func(val []byte) error {
				return proto.Unmarshal(val, info)
			}); err != nil {
				return err
			}
			txid := keyArr[1]
			index, err := strconv.Atoi(keyArr[2])
			if err != nil {
				return fmt.Errorf("invalid key:%s", ukey)
			}

			if info.Address != address {
				return fmt.Errorf("data anomalies key:%s value:%v", ukey, info)
			}
			utxos = append(utxos, &model.UTXO{
				TxID:  txid,
				Index: index,
				Value: fmt.Sprintf("%.8f", info.Value),
			})
		}

		reply.Utxos = utxos
		return nil
	})

	return reply, err
}

// store 存储
func (db *BadgerDB) Store(vins []model.In, vouts []model.Out, lastHeight int64) error {
	start := time.Now()

	utxom, abm, aum, err := db.parseUtxo(vins, vouts)
	if err != nil {
		db.logger.Fatal("parseUtxo", zap.Error(err))
	}

	//store
	if err := db.batchStore(utxom, abm, aum); err != nil {
		db.logger.Fatal("batchStore", zap.Error(err))
	}

	if err := db.storeLastHeight(lastHeight); err != nil {
		db.logger.Fatal("storeLastHeight", zap.Error(err))
	}

	db.logger.Info("Store::Info",
		zap.Int64("lastHeight", lastHeight),
		zap.Int("vout_len", len(vouts)),
		zap.Int("vin_len", len(vins)),
		zap.Duration("ttl", time.Since(start)))
	return nil
}

func (db *BadgerDB) storeLastHeight(lastHeight int64) error {
	if err := db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(StoreHeight), pkg.Int64ToBytes(lastHeight))
	}); err != nil {
		return err
	}
	return nil
}

func (db *BadgerDB) parseUtxo(vins []model.In, vouts []model.Out) (map[string]*UtxoInfo, map[string]decimal.Decimal, map[string]*strset.Set, error) {
	um := make(map[string]*UtxoInfo, defaultMapCap)
	abm := make(map[string]decimal.Decimal, defaultMapCap) //存储地址金额变动
	aaum := make(map[string]*strset.Set, defaultMapCap)    //存储地址下面新增utxo集合
	adum := make(map[string]*strset.Set, defaultMapCap)    //存储地址下面移除utxo集合
	am := make(map[string]struct{}, defaultMapCap)         //地址
	needSearchInfoKeys := make([]string, 0, defaultMapCap)
	for _, vout := range vouts {
		um[vout.UKey] = &UtxoInfo{
			Address: vout.Address,
			Value:   vout.Value,
		}
		am[vout.Address] = struct{}{}

		//新增地址utxo集合
		addAddressUtxo(aaum, vout.Address, vout.UKey)
		//地址余额变动处理
		updateBalance(abm, vout.Address, vout.Value)
	}

	for _, vin := range vins {
		if ui, ok := um[vin.UKey]; ok {
			ui.Spend = &Spend{
				Txid:  vin.Spend.TxID,
				Index: uint32(vin.Spend.Index),
			}
			um[vin.UKey] = ui

			//移除地址utxo集合
			addAddressUtxo(adum, ui.Address, vin.UKey)
			//地址余额变动处理
			updateBalance(abm, ui.Address, -ui.Value)
		} else {
			//已花费 待查询地址金额
			um[vin.UKey] = &UtxoInfo{
				Spend: &Spend{
					Txid:  vin.Spend.TxID,
					Index: uint32(vin.Spend.Index),
				},
			}
			needSearchInfoKeys = append(needSearchInfoKeys, vin.UKey)
		}
	}

	//查询utxo
	if err := db.View(func(txn *badger.Txn) error {
		for _, key := range needSearchInfoKeys {
			item, err := txn.Get([]byte(key))
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}
			if err != badger.ErrKeyNotFound {
				info := &UtxoInfo{}
				if err = item.Value(func(val []byte) error {
					return proto.Unmarshal(val, info)
				}); err != nil {
					return err
				}
				ui := um[key]
				ui.Address = info.Address
				ui.Value = info.Value
				um[key] = ui

				//移除地址utxo集合
				addAddressUtxo(adum, ui.Address, key)
				//地址余额变动处理
				updateBalance(abm, ui.Address, -ui.Value)
				am[ui.Address] = struct{}{}
			}

		}
		return nil
	}); err != nil {
		return nil, nil, nil, err
	}

	//查询余额 地址下utxo集合
	if err := db.View(func(txn *badger.Txn) error {
		for addr := range am {
			//余额
			bitem, err := txn.Get([]byte(addressBalanceKeyPrefix + addr))
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}
			if err != badger.ErrKeyNotFound {
				var value float64
				if err := bitem.Value(func(val []byte) error {
					pf, err := strconv.ParseFloat(string(val), 64)
					if err != nil {
						return err
					}
					value = pf
					return nil
				}); err != nil {
					return err
				}
				updateBalance(abm, addr, value)
			}

			//utxo
			uitem, err := txn.Get([]byte(addressUtxoKeyPrefix + addr))
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}

			if err != badger.ErrKeyNotFound {
				ss := &StringSet{}
				if err := uitem.Value(func(val []byte) error {
					return proto.Unmarshal(val, ss)
				}); err != nil {
					return err
				}
				aaum[addr] = mergeUtxoSet(strset.New(ss.Members...), aaum[addr], adum[addr])
			} else {
				aaum[addr] = mergeUtxoSet(strset.New(), aaum[addr], adum[addr])
			}
		}
		return nil
	}); err != nil {
		return nil, nil, nil, err
	}

	return um, abm, aaum, nil
}

func (db *BadgerDB) batchStore(um map[string]*UtxoInfo, abm map[string]decimal.Decimal, aum map[string]*strset.Set) error {
	// 创建一个WriteBatch
	wb := db.NewWriteBatch()
	defer wb.Cancel()

	for key, info := range um {
		b, err := proto.Marshal(info)
		if err != nil {
			return err
		}
		if err := wb.Set([]byte(key), b); err != nil {
			return err
		}
	}

	for addr, amount := range abm {
		key := addressBalanceKeyPrefix + addr
		if err := wb.Set([]byte(key), []byte(amount.StringFixed(8))); err != nil {
			return err
		}
	}

	for addr, set := range aum {
		key := addressUtxoKeyPrefix + addr
		ss := &StringSet{
			Members: set.List(),
		}
		b, err := proto.Marshal(ss)
		if err != nil {
			return err
		}
		if err := wb.Set([]byte(key), b); err != nil {
			return err
		}
	}

	// 提交WriteBatch，将数据写入数据库
	if err := wb.Flush(); err != nil {
		return err
	}
	return nil
}

func updateBalance(abm map[string]decimal.Decimal, address string, value float64) {
	if bal, ok := abm[address]; !ok {
		abm[address] = decimal.NewFromFloat(value)
	} else {
		abm[address] = bal.Add(decimal.NewFromFloat(value))
	}
}

func addAddressUtxo(asm map[string]*strset.Set, address string, key string) {
	if set, ok := asm[address]; !ok {
		asm[address] = strset.New(key)
	} else {
		set.Add(key)
		asm[address] = set
	}
}

func mergeUtxoSet(base *strset.Set, add *strset.Set, del *strset.Set) *strset.Set {
	if add != nil {
		base.Add(add.List()...)
	}
	if del != nil {
		base.Remove(del.List()...)
	}
	return base
}
