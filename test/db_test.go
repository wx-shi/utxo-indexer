package test

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dgraph-io/badger/v3"
	"github.com/shopspring/decimal"

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
					continue
				}
				amount = amount.Add(decimal.NewFromFloat(u.Value))
			}

			if a1.StringFixed(8) != amount.StringFixed(8) {
				t.Fatalf("%s 余额异常", addr)
			}
			t.Logf("%s %s %s\n", addr, a1.StringFixed(8), amount.StringFixed(8))
		}
		return nil
	})
}

func TestAddress(t *testing.T) {
	address, err := GetAddressByScriptPubKeyResult(btcjson.ScriptPubKeyResult{
		Asm:     "040a464653204c756b652d4a72206c656176652074686520626c6f636b636861696e20616c6f6e65210a4f682c20616e6420676f642069736e2774207265616c0a OP_CHECKSIG",
		Hex:     "41040a464653204c756b652d4a72206c656176652074686520626c6f636b636861696e20616c6f6e65210a4f682c20616e6420676f642069736e2774207265616c0aac",
		ReqSigs: 0,
		Type:    "pubkey",
	})

	t.Log(address, err)

}
func GetAddressByScriptPubKeyResult(sp btcjson.ScriptPubKeyResult) (string, error) {
	switch sp.Type {
	case "pubkey":
		data, err := hex.DecodeString(sp.Hex[2 : len(sp.Hex)-2])
		if err != nil {
			return "", err
		}
		addr, err := btcutil.NewAddressPubKey(data, &chaincfg.MainNetParams)
		if err != nil {
			return "", err
		}
		return addr.EncodeAddress(), err
	}
	return "", nil
}

// func TestBlock(t *testing.T) {
// 	// Initialize Bitcoin JSON-RPC client
// 	btcClient, err := rpcclient.New(&rpcclient.ConnConfig{
// 		Host:         "192.168.0.21:8332",
// 		User:         "btc",
// 		Pass:         "btc2022",
// 		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
// 		DisableTLS:   true, // Bitcoin core does not provide TLS by default
// 	}, nil)
// 	if err != nil {
// 		logger.Fatal("Error initializing Bitcoin RPC client", zap.Error(err))
// 	}

// 	hash, _ := btcClient.GetBlockHash(486194)
// 	res1, _ := btcClient.GetBlockVerbose(hash)

// 	res, _ := btcClient.GetBlockVerboseTx(hash)

// 	if len(res.Tx) == len(res1.Tx) {
// 		fmt.Println("一样")
// 	}
// 	for _, tx := range res.Tx {
// 		if len(tx.BlockHash) > 0 {
// 			fmt.Println(tx.Txid)
// 		}
// 	}
// }
