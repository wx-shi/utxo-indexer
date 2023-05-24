package db

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/golang/protobuf/proto"
	"github.com/scylladb/go-set/strset"
)

func sadd(txn *badger.Txn, setKey string, members ...string) error {
	item, err := txn.Get([]byte(setKey))
	if err != nil && err != badger.ErrKeyNotFound {
		return err
	}

	set := strset.New(members...)

	if err != badger.ErrKeyNotFound {
		value, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		stringSet := &StringSet{}
		err = proto.Unmarshal(value, stringSet)
		if err != nil {
			return err
		}

		set.Add(stringSet.Members...)
	}

	setData, err := proto.Marshal(&StringSet{Members: set.List()})
	if err != nil {
		return err
	}

	return txn.Set([]byte(setKey), setData)
}

func smembers(txn *badger.Txn, setKey string) ([]string, error) {
	item, err := txn.Get([]byte(setKey))
	if err != nil {
		return nil, err
	}

	value, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	stringSet := &StringSet{}
	err = proto.Unmarshal(value, stringSet)
	if err != nil {
		return nil, err
	}

	return stringSet.Members, nil
}

func srem(txn *badger.Txn, setKey string, members ...string) error {
	item, err := txn.Get([]byte(setKey))
	if err != nil {
		return err
	}

	value, err := item.ValueCopy(nil)
	if err != nil {
		return err
	}

	stringSet := &StringSet{}
	err = proto.Unmarshal(value, stringSet)
	if err != nil {
		return err
	}

	set := strset.New(stringSet.Members...)
	set.Remove(members...)

	setData, err := proto.Marshal(&StringSet{Members: set.List()})
	if err != nil {
		return err
	}

	return txn.Set([]byte(setKey), setData)
}
