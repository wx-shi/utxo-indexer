package test

import (
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
	us, amout, err := badgerDB.GetUTXOByAddress("15VF3MsCzjHmFQ3wK3SMrTEBTmFY8zhJnU")
	t.Logf("debug:%v %v %v", us, amout, err)

}
