# UTXO Indexer
该项目用于查询BTC地址余额、UTXO,可快速将btc utxo数据存储完成,存储使用了badger db

# 适用场景
- BTC节点只用于数据查询，不能使用节点自带钱包工具
- 想要查询节点外部地址余额或者UTXO


# key定义

| key          | value          | 是否实现|
|--------------|----------------| ---|
| u:txid:index | 存储 UTXO 信息，包括关联的地址、金额、区块高度以及消费此 UTXO 的交易信息（如果已消费)  |✅|
| au:address   | 存储与特定地址关联的 UTXO 列表（使用 txid:index 格式）            |✅|
| ab:address   | 存储特定地址的总金额           |✅|

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
## /height
- request
```
{}
```

- response
```
{
    "code": 200,
    "data": {
        "store_height": 431558,
        "node_height": 791173
    }
}
```

## /utxo 
- request
```
{
    "address": "1rEVUiXmfgXbfePBQJZhvuHbyYWEw86TL",
    "page_size": 10,
    "page": 0
}
```
- reply
```
{
    "code": 200,
    "data": {
        "balance": "75.67499846",
        "page": 0,
        "page_size": 10,
        "total_size": 103,
        "utxos": [
            {
                "tx_id": "ce0d6c2b7a963484d3a1c8b25460bb1516dce7acb59c78453f1a602765319c82",
                "index": 1,
                "value": "0.00190000"
            },
            {
                "tx_id": "28c2ab1ca8467d2abee113879d515b260043319979d8dca8ee985ed87a66b236",
                "index": 1,
                "value": "0.08611817"
            },
            {
                "tx_id": "c3f54a27494087451e09c3397a45e184cd8ce0561b06de0bc6d1b17e9e9dae3b",
                "index": 1,
                "value": "0.00956249"
            },
            {
                "tx_id": "db12a4477cf165cc431f45ad0f73e470e4037c5441e11065a6bfb3f718b70d00",
                "index": 1,
                "value": "0.00549172"
            },
            {
                "tx_id": "8c296522f8099cc61f3b357fe2e9304068a7a365d3bb27941d9f01a6e1c918b0",
                "index": 1,
                "value": "0.00759057"
            },
            {
                "tx_id": "c8b31b1b6ea0c9346ebbf0bdc90da66f2e404a6867f5ec0724355ca7ee8489ae",
                "index": 1,
                "value": "0.00337786"
            },
            {
                "tx_id": "7dcbe4324b35419aeee9a665985658968944f3f68c476a07bf247907612ab338",
                "index": 1,
                "value": "0.01351213"
            },
            {
                "tx_id": "b54fc142ce73022d5b6f0a9b355e3cb433001a8551978af1d3f3a082d8dc6de2",
                "index": 1,
                "value": "0.02993500"
            },
            {
                "tx_id": "cda51f8134d3afaabb1bd5702402c6f76a285b2a4eb4f9312e84a56ce17dc137",
                "index": 1,
                "value": "0.03100000"
            },
            {
                "tx_id": "3103fa33c2bb943a8246daba24b1b0c1d90f69fe53bc22c84474bd50db6d3b63",
                "index": 1,
                "value": "0.00188605"
            }
        ]
    }
}
```