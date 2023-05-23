package test

import (
	"fmt"
	"github.com/shopspring/decimal"
	"strconv"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/golang/protobuf/proto"
	"github.com/wx-shi/utxo-indexer/internal/config"
	"github.com/wx-shi/utxo-indexer/internal/db"
	"github.com/wx-shi/utxo-indexer/pkg"
)

var (
	logger, _   = pkg.NewLogger("info")
	badgerDB, _ = db.NewBadgerDB(&config.BadgerDBConfig{
		Directory: "../tmp",
	}, logger)
)

func TestHeight(t *testing.T) {
	defer logger.Sync()
	defer badgerDB.Close()

	height, err := badgerDB.GetStoreHeight()
	t.Logf("%d %v", height, err)
}

func TestUtxo(t *testing.T) {
	info := &db.UtxoInfo{}
	err := badgerDB.View(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte("u:282b861e411dc3b61aa06e9e13abf49bce5c571e21a19c37f738244cee33b778:0"))
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}

		if err != badger.ErrKeyNotFound {
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			return proto.UnmarshalMerge(val, info)
		}

		return nil
	})
	t.Logf("debug:%+v %v", info, err)
}

func TestAmount(t *testing.T) {
	badgerDB.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("ab:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			a1 := decimal.Decimal{}
			err := item.Value(func(v []byte) error {
				pf, err := strconv.ParseFloat(string(v), 64)
				if err != nil {
					return err
				}
				a1 = decimal.NewFromFloat(pf)
				return nil
			})
			if err != nil {
				return err
			}

			addr := strings.Split(string(k), ":")[1]
			aukey := "au:" + addr
			auitem, err := txn.Get([]byte(aukey))
			if err != nil {
				return err
			}
			auval, err := auitem.ValueCopy(nil)
			if err != nil {
				return err
			}
			ls := &db.StringSet{}
			if err := proto.Unmarshal(auval, ls); err != nil {
				return err
			}

			amount := decimal.Decimal{}
			for _, uk := range ls.Members {
				ui, err := txn.Get([]byte(uk))
				if err != nil {
					return err
				}
				uv, err := ui.ValueCopy(nil)
				if err != nil {
					return err
				}
				u := &db.UtxoInfo{}
				if err := proto.Unmarshal(uv, u); err != nil {
					return err
				}
				if u.Spend != nil {
					fmt.Println("spend")
					continue
				}
				amount = amount.Add(decimal.NewFromFloat(u.Value))
			}

			if a1.StringFixed(8) != amount.StringFixed(8) {
				panic(fmt.Sprintf("%s 余额异常", addr))
			}
			fmt.Printf("%s %s %s\n", addr, a1.StringFixed(8), amount.StringFixed(8))
		}
		return nil
	})
}
