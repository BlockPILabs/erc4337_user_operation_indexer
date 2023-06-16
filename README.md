# erc4337_user_operation_indexer

## Build
```bash
make
```

## Usage
### memory db
```bash
./build/indexer \
  --chain polygon-mumbai  \
  --backend https://polygon-mumbai.blockpi.network/v1/rpc/{APIKEY}  \
  --db.engin memory
```

### redis
```bash
./build/indexer \
  --chain polygon-mumbai \
  --backend https://polygon-mumbai.blockpi.network/v1/rpc/{APIKEY} \
  --db.engin redis \
  --db.ds "redis://passwd@127.0.0.1:6379"
```

### pebble
```bash
./build/indexer \
  --chain polygon-mumbai \
  --backend https://polygon-mumbai.blockpi.network/v1/rpc/{APIKEY} \
  --db.engin pebble \
  --db.ds "data/db"
```

## help
```bash
   --listen value       listen (default: "127.0.0.1:2052")
   --chain value        Chain
   --entrypoint value   Entrypoint contract (default: "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789")
   --backend value      Backend chain rpc provider url
   --db.prefix value    Backing database prefix
   --db.engin value     Backing database implementation to use ('memory' or 'redis' or 'pebble') (default: "memory")
   --db.ds value        mysql://user:passwd@hostname:port/databasename, redis://passwd@host:port
   --block.start value  {"arbitrum":79305493,"arbitrum-goerli":17068300,"arbitrum-nova":8945015,"ethereum":17066994,"ethereum-goerli":8812127,"ethereum-sepolia":3296058,"optimism":93335977,"optimism-goerli":10442160,"polygon":41402415,"polygon-mumbai":34239265} (default: 0)
   --block.range value  eth_getLogs block range (default: 1000)
   --help, -h           show help

```

## API
### eth_getLogsByUserOperation
Parameters: Array - User operation hash
```bash
curl 'http://127.0.0.1:2052' \
-X POST -H "Content-Type: application/json" \
--data '{
    "jsonrpc": "2.0",
    "method": "eth_getLogsByUserOperation",
    "params": [
        "0xaa6f620266962dbed7778bff708be6891d92935ba1b6120781aca1aa37f9c560",
        "0xcf8b2943927b6b905e5d3c870d19ff7cbfc8bce6c5fd3e59581cebe51f3400c1"
    ],
    "id": 1
}'

{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        {LOGS1},
        {LOGS2}
    ]
}
```
