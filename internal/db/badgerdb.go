package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/shopspring/decimal"
	"github.com/wx-shi/utxo-indexer/internal/config"
	"github.com/wx-shi/utxo-indexer/internal/model"
	"github.com/wx-shi/utxo-indexer/pkg"
	"go.uber.org/zap"
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
		WithDetectConflicts(false).     // 如果不需要事务冲突检测，禁用它以提高写入性能
		WithCompression(options.ZSTD).  //使用Snappy压缩以减少存储空间
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

const StoreHeight = "store::height"

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

func (db *BadgerDB) GetUTXOByAddress(address string, skipUse bool) ([]model.UTXO, string, error) {
	utxos := make([]model.UTXO, 0, 10)
	var amonut decimal.Decimal

	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(fmt.Sprintf("addr:%s:", address))

		//前缀查询 addr:address:
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			use, err := pkg.BytesToBool(v)
			if err != nil {
				return err
			}
			if use && skipUse {
				continue
			}
			var (
				utxoKey string
				txid    string
				index   int
			)

			if arr := strings.Split(string(k), string(prefix)); len(arr) != 2 {
				return fmt.Errorf("invalid key:%s", k)
			} else {
				if hi := strings.Split(arr[1], ":"); len(hi) != 2 {
					return fmt.Errorf("invalid key:%s", k)
				} else {
					txid = hi[0]
					id, err := strconv.Atoi(hi[1])
					if err != nil {
						return fmt.Errorf("invalid key:%s", k)
					}
					index = id
				}

				utxoKey = fmt.Sprintf("utxo:%s", arr[1])
			}

			uitem, err := txn.Get([]byte(utxoKey))
			if err != nil {
				return err
			}

			uvalue, err := uitem.ValueCopy(nil)
			if err != nil {
				return err
			}

			if uarr := strings.Split(string(uvalue), ":"); len(uarr) != 2 {
				return fmt.Errorf("invalid value:%s", uvalue)
			} else {
				if uarr[0] != address {
					return fmt.Errorf("invalid value:%s", uvalue)
				}
				a, err := strconv.ParseFloat(uarr[1], 10)
				if err != nil {
					return fmt.Errorf("invalid value:%s", uvalue)
				}

				utxos = append(utxos, model.UTXO{
					Hash:    txid,
					Index:   index,
					Address: address,
					Value:   a,
					Spend:   use,
				})
				if !use {
					amonut = amonut.Add(decimal.NewFromFloat(a))
				}
			}
		}

		return nil
	})

	return utxos, amonut.StringFixed(8), err
}

// store 存储
func (db *BadgerDB) Store(vins []model.UseUTXO, vouts []model.UTXO, lastHeight int64) error {
	start := time.Now()
	utxoMap := make(map[string]model.UTXO, 10000) //utxo:txid:index -- utxo
	for _, vout := range vouts {
		utxoMap[fmt.Sprintf("%s:%d", vout.Hash, vout.Index)] = vout
	}

	//已使用utxo
	useUtxoKeys := make([]string, 0, len(vins)) //已使用utxo key utxo:txid:index
	for _, vin := range vins {
		key := fmt.Sprintf("%s:%d", vin.Txid, vin.Vout)
		if utxo, ok := utxoMap[key]; ok {
			utxo.Spend = true //置为已花费
			utxoMap[key] = utxo
		} else {
			useUtxoKeys = append(useUtxoKeys, key) //历史已花费key
		}
	}

	useAddrUtxoKeys, err := db.searchUseAddrUtxoKeys(useUtxoKeys)
	if err != nil {
		db.logger.Fatal("searchUseAddrUtxoKeys", zap.Error(err))
	}

	if err := db.batchStoreUtxo(utxoMap, useAddrUtxoKeys); err != nil {
		db.logger.Fatal("batchStoreUtxo", zap.Error(err))
	}

	//存储最后高度
	if err := db.storeLastHeight(err, lastHeight); err != nil {
		db.logger.Fatal("storeLastHeight", zap.Error(err))
	}

	if err := db.RunValueLogGC(0.1); err != nil &&
		err != badger.ErrNoRewrite && err != badger.ErrRejected {
		db.logger.Fatal("RunValueLogGC", zap.Error(err))
	}

	db.logger.Info("Store::Info",
		zap.Int64("lastHeight", lastHeight),
		zap.Int("vout_len", len(vouts)),
		zap.Int("vin_len", len(vins)),
		zap.Duration("ttl", time.Since(start)))
	return nil
}

func (db *BadgerDB) storeLastHeight(err error, lastHeight int64) error {
	if err = db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(StoreHeight), pkg.Int64ToBytes(lastHeight))
	}); err != nil {
		return err
	}
	return nil
}

func (db *BadgerDB) batchStoreUtxo(utxoMap map[string]model.UTXO, useAddrUtxoKeys []string) error {
	// 创建一个WriteBatch
	wb := db.NewWriteBatch()
	defer wb.Cancel()

	for key, val := range utxoMap {
		// 新增utxo
		if err := wb.Set([]byte(fmt.Sprintf("utxo:%s", key)), []byte(fmt.Sprintf("%s:%.8f", val.Address, val.Value))); err != nil {
			return err
		}

		// 新增地址与utxo绑定关系
		if err := wb.Set([]byte(fmt.Sprintf("addr:%s:%s", val.Address, key)), pkg.BoolToBytes(val.Spend)); err != nil {
			return err
		}
	}

	for _, key := range useAddrUtxoKeys {
		if err := wb.Set([]byte(key), pkg.BoolToBytes(true)); err != nil {
			return err
		}
	}

	// 提交WriteBatch，将数据写入数据库
	if err := wb.Flush(); err != nil {
		return err
	}
	return nil
}

func (db *BadgerDB) searchUseAddrUtxoKeys(useUtxoKeys []string) ([]string, error) {
	useAddrUtxoKeys := make([]string, 0, len(useUtxoKeys))
	if len(useUtxoKeys) > 0 {
		// 历史已花费处理
		if err := db.View(func(txn *badger.Txn) error {
			for _, utxo := range useUtxoKeys {
				key := []byte(fmt.Sprintf("utxo:%s", utxo))
				item, err := txn.Get(key)
				if err != nil {
					if err == badger.ErrKeyNotFound {
						continue
					}
					return err
				}
				value, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}

				var address string
				if arr := strings.Split(string(value), ":"); len(arr) != 2 {
					return fmt.Errorf("invalid value:%s", value)
				} else {
					address = arr[0]
				}
				useAddrUtxoKeys = append(useAddrUtxoKeys, fmt.Sprintf("addr:%s:%s", address, utxo))
			}

			return nil
		}); err != nil {
			return nil, err
		}
	}
	return useAddrUtxoKeys, nil
}
