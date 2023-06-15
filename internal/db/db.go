package db

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"strconv"
	"strings"
	"time"

	tmdb "github.com/cosmos/cosmos-db"
	"github.com/scylladb/go-set/strset"
	"github.com/shopspring/decimal"
	"github.com/wx-shi/utxo-indexer/internal/config"
	"github.com/wx-shi/utxo-indexer/internal/model"
	"github.com/wx-shi/utxo-indexer/pkg"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	utxoKeyPrefix           = "u:"
	addressBalanceKeyPrefix = "ab:"
	addressUtxoKeyPrefix    = "au:"
	StoreHeight             = "s:h"
	defaultMapCap           = 10000

	udbName  = "utxo"
	bdbName  = "balance"
	audbName = "address_utxo"
)

type DB struct {
	udb    tmdb.DB
	bdb    tmdb.DB
	audb   tmdb.DB
	logger *zap.Logger
}

func NewDB(conf *config.DBConfig, logger *zap.Logger) (*DB, error) {
	udb, err := tmdb.NewDB(udbName, tmdb.BackendType(conf.DBType), conf.Dir)
	if err != nil {
		return nil, err
	}
	bdb, err := tmdb.NewDB(bdbName, tmdb.BackendType(conf.DBType), conf.Dir)
	if err != nil {
		return nil, err
	}

	audb, err := tmdb.NewDB(audbName, tmdb.BackendType(conf.DBType), conf.Dir)
	if err != nil {
		return nil, err
	}

	return &DB{
		udb:    udb,
		bdb:    bdb,
		audb:   audb,
		logger: logger,
	}, nil
}

func (db *DB) Close() error {
	g, _ := errgroup.WithContext(context.Background())
	g.Go(db.udb.Close)
	g.Go(db.bdb.Close)
	g.Go(db.audb.Close)
	return g.Wait()
}

func (db *DB) GetStoreHeight() (int64, error) {
	val, err := db.udb.Get([]byte(StoreHeight))
	if err != nil {
		return 0, err
	}
	if len(val) == 0 {
		return 0, nil
	}
	return pkg.BytesToInt64(val), err
}

func (db *DB) GetUTXOByAddress(address string, page int, pageSize int) (*model.UTXOReply, error) {

	abKey := addressBalanceKeyPrefix + address
	auKey := addressUtxoKeyPrefix + address

	reply := &model.UTXOReply{
		Page:     page,
		PageSize: pageSize,
	}

	// 获取余额
	{
		val, err := db.bdb.Get([]byte(abKey))
		if err != nil {
			return nil, err
		}
		var bal float64
		if len(val) == 0 {
			reply.Balance = fmt.Sprintf("%.8f", bal)
			return reply, nil
		}
		bal, err = strconv.ParseFloat(string(val), 64)
		if err != nil {
			return nil, err
		}
		reply.Balance = fmt.Sprintf("%.8f", bal)

	}

	// 获取utxo列表
	us := &StringSet{}
	{
		uitem, err := db.audb.Get([]byte(auKey))
		if err != nil {
			return nil, err
		}
		if len(uitem) == 0 {
			reply.TotalSize = 0
			return reply, nil
		}

		if err := proto.Unmarshal(uitem, us); err != nil {
			return nil, err
		}
	}

	// 获取utxo列表
	reply.TotalSize = len(us.Members)
	uarr := pkg.Paginate(us.Members, page, pageSize)

	utxos := make([]*model.UTXO, 0, len(uarr))
	for _, ukey := range uarr {
		keyArr := strings.Split(ukey, ":")
		if len(keyArr) != 3 {
			return nil, fmt.Errorf("invalid key:%s", ukey)
		}
		val, err := db.udb.Get([]byte(ukey))
		if err != nil {
			return nil, err
		}
		info := &UtxoInfo{}
		if err := proto.Unmarshal(val, info); err != nil {
			return nil, err
		}
		txid := keyArr[1]
		index, err := strconv.Atoi(keyArr[2])
		if err != nil {
			return nil, fmt.Errorf("invalid key:%s", ukey)
		}

		if info.Address != address {
			return nil, fmt.Errorf("data anomalies key:%s value:%v", ukey, info)
		}
		utxos = append(utxos, &model.UTXO{
			TxID:  txid,
			Index: index,
			Value: fmt.Sprintf("%.8f", info.Value),
		})
	}

	reply.Utxos = utxos

	return reply, nil
}

func (db *DB) GetUTXOInfoByKeys(keys []string) (model.UTXOInfoReply, error) {
	reply := make(model.UTXOInfoReply, len(keys))

	for i := 0; i < len(keys); i++ {
		key := keys[i]
		val, err := db.udb.Get([]byte(utxoKeyPrefix + key))
		if err != nil {
			return nil, err
		}
		if len(val) == 0 {
			reply[key] = nil
		} else {
			info := &UtxoInfo{}
			if err := proto.Unmarshal(val, info); err != nil {
				return nil, err
			}
			reply[key] = &model.UtxoInfo{
				Address: info.Address,
				Value:   info.Value,
			}
		}
	}

	return reply, nil
}

// store 存储
func (db *DB) Store(vins []model.In, vouts []model.Out, lastHeight int64) error {
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

func (db *DB) parseUtxo(vins []model.In, vouts []model.Out) (map[string]*UtxoInfo, map[string]decimal.Decimal, map[string]*strset.Set, error) {
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
	for _, key := range needSearchInfoKeys {
		val, err := db.udb.Get([]byte(key))
		if err != nil {
			return nil, nil, nil, err
		}
		if len(val) > 0 {
			info := &UtxoInfo{}
			if err := proto.Unmarshal(val, info); err != nil {
				return nil, nil, nil, err
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

	//查询余额 地址下utxo集合
	for addr := range am {
		//余额
		{
			bval, err := db.bdb.Get([]byte(addressBalanceKeyPrefix + addr))
			if err != nil {
				return nil, nil, nil, err
			}
			var value float64
			if len(bval) > 0 {
				value, err = strconv.ParseFloat(string(bval), 64)
				if err != nil {
					return nil, nil, nil, err
				}
				updateBalance(abm, addr, value)
			}
		}

		//utxo
		{
			uval, err := db.audb.Get([]byte(addressUtxoKeyPrefix + addr))
			if err != nil {
				return nil, nil, nil, err
			}
			if len(uval) > 0 {
				ss := &StringSet{}
				if err := proto.Unmarshal(uval, ss); err != nil {
					return nil, nil, nil, err
				}
				aaum[addr] = mergeUtxoSet(strset.New(ss.Members...), aaum[addr], adum[addr])
			} else {
				aaum[addr] = mergeUtxoSet(strset.New(), aaum[addr], adum[addr])
			}
		}
	}

	return um, abm, aaum, nil
}

func (db *DB) batchStore(um map[string]*UtxoInfo, abm map[string]decimal.Decimal, aum map[string]*strset.Set) error {
	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		// 创建一个WriteBatch
		wb := db.udb.NewBatch()
		defer wb.Close()

		for key, info := range um {
			b, err := proto.Marshal(info)
			if err != nil {
				return err
			}
			if err := wb.Set([]byte(key), b); err != nil {
				return err
			}
		}
		// 提交WriteBatch，将数据写入数据库
		return wb.WriteSync()
	})
	g.Go(func() error {
		wb := db.bdb.NewBatch()
		defer wb.Close()

		for addr, amount := range abm {
			key := addressBalanceKeyPrefix + addr

			if amount.IsZero() {
				if err := wb.Delete([]byte(key)); err != nil {
					return err
				}
			} else {
				if err := wb.Set([]byte(key), []byte(amount.StringFixed(8))); err != nil {
					return err
				}
			}
		}
		// 提交WriteBatch，将数据写入数据库
		return wb.WriteSync()
	})
	g.Go(func() error {
		wb := db.audb.NewBatch()
		defer wb.Close()

		for addr, set := range aum {
			key := addressUtxoKeyPrefix + addr
			if len(set.List()) == 0 {
				if err := wb.Delete([]byte(key)); err != nil {
					return err
				}
			} else {
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
		}

		// 提交WriteBatch，将数据写入数据库
		return wb.WriteSync()
	})

	return g.Wait()
}

func (db *DB) storeLastHeight(lastHeight int64) error {
	return db.udb.SetSync([]byte(StoreHeight), pkg.Int64ToBytes(lastHeight))
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
