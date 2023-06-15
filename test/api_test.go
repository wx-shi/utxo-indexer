package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/guonaihong/gout"
	"github.com/shopspring/decimal"
	"github.com/wx-shi/utxo-indexer/internal/model"
)

type commonRepley struct {
	Code int             `json:"code,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
	Msg  string          `json:"msg,omitempty"`
}

func TestApiUTXO(t *testing.T) {
	utxoLen := 0
	totalSize := 0
	balance := decimal.Decimal{}
	balance2 := decimal.Decimal{}
	for i := 0; ; i++ {
		ur, err := getUtxo("14cZMQk89mRYQkDEj8Rn25AnGoBi5H6uer", i, 100)
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(ur.Utxos) == 0 {
			totalSize = ur.TotalSize
			v, _ := decimal.NewFromString(ur.Balance)
			balance2 = v
			break
		}
		utxoLen += len(ur.Utxos)
		for i := 0; i < len(ur.Utxos); i++ {
			u := ur.Utxos[i]
			v, _ := decimal.NewFromString(u.Value)
			balance = balance.Add(v)
		}
	}

	fmt.Println(utxoLen, totalSize)
	fmt.Println(balance.StringFixed(8), balance2.StringFixed(8))
}

func BenchmarkApiHeight(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := getHeight()
		if err != nil {
			b.Error(err)
		}
	}
}

func getHeight() (int64, error) {
	reply := &commonRepley{}
	if err := gout.POST("http://192.168.0.21:3000/height").BindJSON(reply).Do(); err != nil {
		fmt.Println(err)
		return 0, err
	}
	hr := &model.HeightReply{}
	err := json.Unmarshal(reply.Data, hr)
	return hr.StoreHeight, err
}

func getUtxo(address string, page, pageSize int) (*model.UTXOReply, error) {
	reply := &commonRepley{}
	req := &model.UTXORequest{
		Address:  address,
		Page:     page,
		PageSize: pageSize,
	}
	if err := gout.POST("http://192.168.0.21:3000/utxo").SetJSON(req).BindJSON(reply).Do(); err != nil {
		fmt.Println(err)
		return nil, err
	}
	ur := &model.UTXOReply{}
	if err := json.Unmarshal(reply.Data, ur); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return ur, nil
}
