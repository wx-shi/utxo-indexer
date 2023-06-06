package db

import (
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
)

const (
	// DefaultBlockCacheSize is 2 GB.
	DefaultBlockCacheSize = 2 << 30

	// DefaultIndexCacheSize is 2 GB.
	DefaultIndexCacheSize = 2 << 30

	// DefaultMaxTableSize is 256 MB. The larger
	// this value is, the larger database transactions
	// storage can handle (~15% of the max table size
	// == max commit size).
	DefaultMaxTableSize = 256 << 20

	// DefaultLogValueSize is 64 MB.
	DefaultLogValueSize = 64 << 20

	// DefaultCompressionMode is the default block
	// compression setting.
	DefaultCompressionMode = options.Snappy

	// Default GC settings for reclaiming
	// space in value logs.
	defaultGCInterval     = 30 * time.Minute
	defualtGCDiscardRatio = 0.1
)

func DefaultBadgerOptions(dir string) badger.Options {
	opts := badger.DefaultOptions(dir)

	// By default, we do not compress the table at all. Doing so can
	// significantly increase memory usage.
	opts.Compression = DefaultCompressionMode

	// Use an extended table size for larger commits.
	opts.MemTableSize = DefaultMaxTableSize
	opts.ValueLogFileSize = DefaultLogValueSize

	// To allow writes at a faster speed, we create a new memtable as soon as
	// an existing memtable is filled up. This option determines how many
	// memtables should be kept in memory.
	opts.NumMemtables = 1

	// Don't keep multiple memtables in memory. With larger
	// memtable size, this explodes memory usage.
	opts.NumLevelZeroTables = 1
	opts.NumLevelZeroTablesStall = 2

	// We don't compact L0 on close as this can greatly delay shutdown time.
	opts.CompactL0OnClose = false

	// This value specifies how much memory should be used by table indices. These
	// indices include the block offsets and the bloomfilters. Badger uses bloom
	// filters to speed up lookups. Each table has its own bloom
	// filter and each bloom filter is approximately of 5 MB. This defaults
	// to an unlimited size (and quickly balloons to GB with a large DB).
	opts.IndexCacheSize = DefaultIndexCacheSize

	// Don't cache blocks in memory. All reads should go to disk.
	opts.BlockCacheSize = DefaultBlockCacheSize

	return opts
}
