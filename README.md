# UTXO Indexer
该项目用于查询BTC地址余额、UTXO,可快速将btc utxo数据存储完成

# 适用场景
- BTC节点只用于数据查询，不能使用节点自带钱包工具
- 想要查询节点外部地址余额或者UTXO


# key定义

| key                     | value          | 描述                         | 是否实现|
|-------------------------|----------------|----------------------------| ---|
| utxo:txid:index         | address:amount | 存储utxo                     |✅|
| addr:addrees:txid:index | 0/1            | 存储地址下有什么utxo，0表示未使用，1表示已使用 |✅|

# 构建运行
```
go build
ulimit -n 100000 && ./utxo-indexer
```

# 配置文件
batch_size是批量存储的阈值(累计达到该值进行存储 len_vin+len_vout),block_chan_buf是在存储是继续拉取block_chan_buf个区块数据;
需要将这两个值合理设置，设置太大会很吃内存
```yaml
server:
  host: 0.0.0.0
  port: 3000

log_level: debug

badger_db:
  directory: ./tmp

rpc:
  url: 127.0.0.1:8332
  user: btc
  password: btc2022

indexer:
  batch_size: 1000000
  block_chan_buf: 1000

```


# 接口
## /utxo 
- request
```
{
    "address":"12c6DSiU4Rq3P4ZxziKxzrL5LmMBrzjrJX"
}
```
- reply
```
{
    "code": 200,
    "data": {
        "balance": "50.00000000",
        "utxos": [
            {
                "hash": "0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098",
                "address": "12c6DSiU4Rq3P4ZxziKxzrL5LmMBrzjrJX",
                "index": 0,
                "value": 50
            }
        ]
    }
}
```