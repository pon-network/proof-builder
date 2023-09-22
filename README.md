# PoN Builder

PoN (Proof-of-Neutrality) Builder is a blockchain service built on top of the Go Ethereum implementation of the Ethereum protocol.

By utilizing the [Official Golang implementation of the Ethereum protocol](https://pkg.go.dev/github.com/ethereum/go-ethereum?tab=doc), PoN Builder inherits all of the security, stability, and reliability of the Ethereum blockchain, while providing additional features to enable highly efficient and profitable block building and transaction processing for the PoN network.

The PoN Builder interacts with the mempool to create optimal bundles of transactions in a block template, with the goal of extracting maximum value from each block by default and the option of including additional private transactions. PoN Builder competes with other builders using relays to acquire block space from validators and ensure inclusion of their execution payloads in proposed blocks.

[![PoN Builder](
<https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667>
)](https://docs.pon.network/)
[![PoN Builder Getting Started](
<https://img.shields.io/badge/Documentation-Docusaurus-green>)](https://docs.pon.network/pon/bguide)

Automated builds are available for stable releases and the unstable master branch. Binary
archives are published at <https://github.com/pon-pbs/bbBuilder>.

## Building the source

For prerequisites and detailed build instructions please read the [Installation Instructions](https://geth.ethereum.org/docs/getting-started/installing-geth).

Building `geth` requires both a Go (version 1.20 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

```shell
make geth
```

or, to build the full suite of utilities:

```shell
make all
```

## Executables

The go-ethereum project comes with several wrappers/executables found in the `cmd`
directory.

|  Command   | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| :--------: | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`geth`** | The main Ethereum CLI client. It is the entry point into the Ethereum network (main-, test- or private net), capable of running as a full node (default), archive node (retaining all historical state) or a light node (retrieving data live). It can be used by other processes as a gateway into the Ethereum network via JSON RPC endpoints exposed on top of HTTP, WebSocket and/or IPC transports. `geth --help` and the [CLI page](https://geth.ethereum.org/docs/fundamentals/command-line-options) for command line options. |
|   `clef`   | Stand-alone signing tool, which can be used as a backend signer for `geth`.                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
|  `devp2p`  | Utilities to interact with nodes on the networking layer, without running a full blockchain.                                                                                                                                                                                                                                                                                                                                                                                                                                       |
|  `abigen`  | Source code generator to convert Ethereum contract definitions into easy-to-use, compile-time type-safe Go packages. It operates on plain [Ethereum contract ABIs](https://docs.soliditylang.org/en/develop/abi-spec.html) with expanded functionality if the contract bytecode is also available. However, it also accepts Solidity source files, making development much more streamlined. Please see our [Native DApps](https://geth.ethereum.org/docs/developers/dapp-developer/native-bindings) page for details.                                  |
| `bootnode` | Stripped down version of our Ethereum client implementation that only takes part in the network node discovery protocol, but does not run any of the higher level application protocols. It can be used as a lightweight bootstrap node to aid in finding peers in private networks.                                                                                                                                                                                                                                               |
|   `evm`    | Developer utility version of the EVM (Ethereum Virtual Machine) that is capable of running bytecode snippets within a configurable environment and execution mode. Its purpose is to allow isolated, fine-grained debugging of EVM opcodes (e.g. `evm --code 60ff60ff --debug run`).                                                                                                                                                                                                                                               |
| `rlpdump`  | Developer utility tool to convert binary RLP ([Recursive Length Prefix](https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp)) dumps (data encoding used by the Ethereum protocol both network as well as consensus wise) to user-friendlier hierarchical representation (e.g. `rlpdump --hex CE0183FFFFFFC4C304050583616263`).                                                                                                                                                                                |

## Running `geth`

Going through all the possible command line flags is out of scope here (please consult our
[CLI Wiki page](https://geth.ethereum.org/docs/fundamentals/command-line-options)),
but we've enumerated a few common parameter combos to get you up to speed quickly
on how you can run your own `geth` instance.

### Hardware Requirements

Minimum:

* CPU with 2+ cores
* 4GB RAM
* 1TB free storage space to sync the Mainnet
* 8 MBit/sec download Internet service

Recommended:

* Fast CPU with 4+ cores
* 16GB+ RAM
* High-performance SSD with at least 1TB of free space
* 25+ MBit/sec download Internet service

## Running `geth with PoN Builder`

### Enabling PoN Builder

To enable the PoN Builder, use the following command:

```shell
geth [GETH_FLAGS] \
--http \
--http.api <api_list_to_enable> \
--builder \
--builder.beacon_endpoints <beacon_endpoints> \
--builder.listen_addr <listen_addr> \
--builder.public_accesspoint <public_accesspoint> \
--builder.relay_endpoint <relay_endpoint> \
--builder.payout_pool_tx_gas <payout_pool_tx_gas> \
--builder.secret_key <secret_key> \
--builder.wallet_private_key <wallet_private_key> \
--builder.rpbs <rpbs_service_base_url> \
--builder.metrics \
--builder.metrics_reset \
--builder.bundles \
--builder.bundles_reset \
--builder.bundles_max_lifetime <bundles_max_lifetime> \
--builder.bundles_max_future <bundles_max_future> \
--builder.submission_end_window <submission_end_window> \
--builder.engine_rate_limit <engine_rate_limit> 
```

Make sure to replace the values enclosed in `<` and `>` with the appropriate values for your configuration.

### PoN Builder Flags

| Flag | Description | Default | Required |
| --- | --- | --- | --- |
| `--http` | Enable the HTTP-RPC server | `false` | Yes |
| `--http.api` | API's offered over the HTTP-RPC interface. To enable builder rpc add `mev` to list | `eth,net,mev` | Yes |
| `--builder` | Enable the PoN Builder | `false` | Yes |
| `--builder.beacon_endpoints` | Beacon node endpoints (comma seperated) | `"http://127.0.0.1:5052"` | Yes |
| `--builder.listen_addr` | Listen address for the PoN Builder service locally | `""` | Yes |
| `--builder.public_accesspoint` | Public accesspoint for remote relay to communicate with the builder | `""` | Yes |
| `--builder.relay_endpoint` | Relay endpoint in the format of `https://<payout_pool_address>@<relay_endpoint>` | `""` | Yes |
| `--builder.payout_pool_tx_gas` | Gas cost for payout pool transactions in a block | `300000` | No |
| `--builder.wallet_private_key` | Wallet private key | `""` | Yes |
| `--builder.secret_key` | BLS Secret key | `""` | Yes |
| `--builder.rpbs` | RPBS Service endpoint | `""` | Yes |
| `--builder.metrics` | Enable metrics | `false` | No |
| `--builder.metrics_reset` | Reset metrics | `false` | No |
| `--builder.bundles` | Enable builder mev bundles service | `false` | No |
| `--builder.bundles_reset` | Reset builder mev bundles service | `false` | No |
| `--builder.bundles_max_lifetime` | Max lifetime in seconds (s) for mev bundles (defaults to 30 days) | `2592000` | No |
| `--builder.bundles_max_future` | Max future in seconds (s) for mev bundles (defaults to 30 days) | `2592000` | No |
| `--builder.submission_end_window` | Time in seconds (s) before the end of any bid period that the builder will finalize the last block bid submission | `2` | No |
| `--builder.engine_rate_limit` | Rate limit in milliseconds (ms) for building blocks from internal engine for given slot (must be greater than 1) | `500` | No |

### Example

For example, to enable the PoN Builder with the following configuration:

* `--http`: `true`
* `--http.api`: `eth,net,mev`
* `--builder.beacon_endpoints`: `http://localhost:5052`
* `--builder.listen_addr`: `:10000`
* `--builder.public_accesspoint`: `https://builder.0xblockswap.com`
* `--builder.relay_endpoint`: `https://0x1234abcd1234abcd1234abcd@relayer.0xblockswap.com`
* `--builder.payout_pool_tx_gas`: `300000`
* `--builder.wallet_private_key`: `0x1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd`
* `--builder.secret_key`: `0x1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd`
* `--builder.rpbs`: `http://localhost:3000`
* `--builder.metrics`: `true`
* `--builder.metrics_reset`: `true`
* `--builder.bundles`: `true`
* `--builder.bundles_reset`: `true`
* `--builder.bundles_max_lifetime`: `2592000`
* `--builder.bundles_max_future`: `2592000`
* `--builder.submission_end_window`: `2`
* `--builder.engine_rate_limit`: `500`

Use the following command where the first few flags are specific to running geth, followed by the PoN Builder flags

```shell
geth --goerli --datadir /home/ubuntu/execution --authrpc.addr localhost --authrpc.port 8551 --authrpc.vhosts localhost \
--http \
--http.api eth,net,mev \
--builder \
--builder.beacon_endpoints http://localhost:5052 \
--builder.listen_addr :10000 \
--builder.public_accesspoint https://builder.0xblockswap.com \
--builder.relay_endpoint https://0x1234abcd1234abcd1234abcd@relayer.0xblockswap.com \
--builder.payout_pool_tx_gas 300000 \
--builder.wallet_private_key 0x1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd \
--builder.secret_key 0x1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd \
--builder.rpbs http://localhost:3000 \
--builder.metrics \
--builder.metrics_reset \
--builder.bundles \
--builder.bundles_reset \
--builder.bundles_max_lifetime 2592000 \
--builder.bundles_max_future 2592000 \
--builder.submission_end_window 2 \
--builder.engine_rate_limit 500
```

## Builder API Endpoints

The Builder software provides several API endpoints for interacting with the builder service:

### Check Status

Endpoint: `GET /eth/v1/builder/status`

This endpoint is used to check the status of the builder service.

### Submit Block Bid

Endpoint: `POST /eth/v1/builder/submit_block_bid`

This endpoint is used to submit a block bid to the builder service which builds a block and submits the block bid to the relay. The request body should be a JSON object containing the following fields:

* `slot`: the block slot for the bid (optional, if not provided the builder will use the current slot)
* `bidAmount`: the bid amount in wei (required)
* `payoutPoolAddress`: the payout pool address for the block bid (optional, if not provided the builder will use the payout pool address provided in the builder configuration for the relays)
* `transactions`: an array of signed transaction RLP encoded bytes representing the transactions to prioritize and include in the block bid (optional)
* `suggestedFeeRecipient`: the suggested fee recipient address for the block (required else defaults to null address)
* `noMempoolTxs`: a boolean value indicating whether to include mempool transactions in the block bid (optional, defaults to false)

Sample scripts are provided within `scripts/` to generate and submit block bids.

### Submit Block Bounty Bid

Endpoint: `POST /eth/v1/builder/submit_block_bounty_bid`

This endpoint is used to submit a bounty block bid to the builder service which builds a block and submits the bounty block bid to the relay. The request body should be a JSON object containing the following fields:

* `slot`: the block slot for the bid (required)
* `bidAmount`: the bid amount in wei (required)
* `payoutPoolAddress`: the payout pool address for the block bid (optional, if not provided the builder will use the payout pool address provided in the builder configuration for the relays)
* `transactions`: an array of signed transaction RLP encoded bytes representing the transactions to prioritize and include in a new the bounty block (optional)
* `suggestedFeeRecipient`: the suggested fee recipient address for the block (required else defaults to null address)
* `noMempoolTxs`: a boolean value indicating whether to include mempool transactions in the block bid (optinal, defaults to false)

### Submit Blinded Block

Endpoint: `POST /eth/v1/builder/blinded_blocks`

This endpoint is used to submit a blinded block to the builder service which processes the blinded beacon block and returns the beacon block to the relay, as well as submits to the chain. The request body should be a JSON object being the signed blinded block containing the following fields:

* `message`: the blinded beacon block
* `signature`: the signature of the blinded beacon block

### Builder Dashboard

Endpoint: `GET /eth/v1/builder/`

This endpoint will render an HTML dashboard that displays information about the builder service, including its syncing status, database entries, and various statistics such as average bid amounts and MEV values.

## Builder-specific RPC Calls

PoN Builder exposes a few RPC calls that are specific to the Builder service. These add additional functionality to the standard geth RPC calls.
All builder rpc calls are sent to the listening address and port set by --http and --http.port flags for geth's rpc

### `mev_sendBundle`

This method is used to send a bundle of transactions to the builder service. The request body should be a JSON object containing the following fields:

```json
{
    "jsonrpc": "2.0",
    "method": "mev_sendBundle",
    "params": [    
        {      
            "txs": ["0x003...", "0x056a.."],    // a list of hex-encoded signed transaction bytes            
            "blockNumber": "550000",        // block number for which this bundle is valid - optional only if minTimestamp and maxTimestamp are set
            "minTimestamp": "0",                  // unix timestamp when this bundle becomes active - optional 
            "maxTimestamp": "1672933616",         // unix timestamp how long this bundle stays valid - optional      
            "revertingTxHashes": ["0xk0d..."]   // list of hashes of possibly reverting txs - optional   
        }  
    ],
    "id": 1
}
```

Successful responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "id": "0x123...", // bundle id
        "inserted_at": "2021-05-12T12:00:00Z", // timestamp when the bundle was inserted into the database
        "bundle_hash": "0x123...", // hash of the bundle
        "txs": ["0x003...", "0x056a.."], // a list of hex-encoded signed transaction bytes
        "block_number": "550000", // block number for which this bundle is valid
        "min_timestamp": "0", // unix timestamp when this bundle becomes active
        "max_timestamp": "1672933616", // unix timestamp how long this bundle stays valid
        "reverting_tx_hashes": ["0xk0d..."], // list of hashes of possibly reverting txs
        "builder_pubkey": "0x123...", // public key of the builder
        "builder_signature": "0x123...", // signature of the bundle by the builder can be verified with the public key
        "bundle_transaction_count": "2", // number of transactions in the bundle
        "bundle_total_gas": "1000000", // total gas of the bundle
        "added": true, // whether the bundle was added to a block
        "error": false, // whether the bundle had an error
        "error_message": "", // error message if the bundle had an error
        "failed_retry_count": "0" // number of times the bundle was retried
    }
}
```

Error responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {
        "code": -3200, // error code,
        "message": "error message", // error message
    }
}
```

### `mev_updateBundle`

This method is used to update a bundle of transactions to the builder service. The request body should be a JSON object containing the following fields:

```json
{
    "jsonrpc": "2.0",
    "method": "mev_updateBundle",
    "params": [    
        {      
            "id": "0x1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd", // bundle id to update
            "txs": ["0x003...", "0x056a.."],    // a list of hex-encoded signed transaction bytes            
            "blockNumber": "550000",        // block number for which this bundle is valid - optional only if minTimestamp and maxTimestamp are set
            "minTimestamp": "0",                  // unix timestamp when this bundle becomes active - optional 
            "maxTimestamp": "1672933616",         // unix timestamp how long this bundle stays valid - optional      
            "revertingTxHashes": ["0xk0d..."]   // list of hashes of possibly reverting txs - optional   
        }  
    ],
    "id": 1
}
```

Successful responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "id": "0x123...", // bundle id
        "inserted_at": "2021-05-12T12:00:00Z", // timestamp when the bundle was inserted into the database
        "bundle_hash": "0x123...", // hash of the bundle
        "txs": ["0x003...", "0x056a.."], // a list of hex-encoded signed transaction bytes
        "block_number": "550000", // block number for which this bundle is valid
        "min_timestamp": "0", // unix timestamp when this bundle becomes active
        "max_timestamp": "1672933616", // unix timestamp how long this bundle stays valid
        "reverting_tx_hashes": ["0xk0d..."], // list of hashes of possibly reverting txs
        "builder_pubkey": "0x123...", // public key of the builder
        "builder_signature": "0x123...", // signature of the bundle by the builder can be verified with the public key
        "bundle_transaction_count": "2", // number of transactions in the bundle
        "bundle_total_gas": "1000000", // total gas of the bundle
        "added": true, // whether the bundle was added to a block
        "error": false, // whether the bundle had an error
        "error_message": "", // error message if the bundle had an error
        "failed_retry_count": "0" // number of times the bundle was retried
    }
}
```

Error responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {
        "code": -3200, // error code,
        "message": "error message", // error message
    }
}
```

### `mev_getBundle`

This method is used to get a bundle of transactions from the builder service. The request body should be a JSON object containing the following fields:

```json
{
    "jsonrpc": "2.0",
    "method": "mev_getBundle",
    "params": [
        {
            "id": "0x1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd" // bundle id to get
        }
    ],
    "id": 1
}
```

Successful responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "id": "0x123...", // bundle id
        "inserted_at": "2021-05-12T12:00:00Z", // timestamp when the bundle was inserted into the database
        "bundle_hash": "0x123...", // hash of the bundle
        "block_number": "550000", // block number for which this bundle is valid
        "min_timestamp": "0", // unix timestamp when this bundle becomes active
        "max_timestamp": "1672933616", // unix timestamp how long this bundle stays valid
        "builder_pubkey": "0x123...", // public key of the builder
        "builder_signature": "0x123...", // signature of the bundle by the builder can be verified with the public key
        "bundle_transaction_count": "2", // number of transactions in the bundle
        "bundle_total_gas": "1000000", // total gas of the bundle
        "added": true, // whether the bundle was added to a block
        "error": false, // whether the bundle had an error
        "error_message": "", // error message if the bundle had an error
        "failed_retry_count": "0" // number of times the bundle was retried
    }
}
```

For privacy and security reasons, the `txs` and `reverting_tx_hashes` fields are not returned when retrieving a bundle.

Error responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {
        "code": -3200, // error code,
        "message": "error message", // error message
    }
}
```

### `mev_cancelBundle`

This method is used to cancel a bundle of transactions to the builder service. The request body should be a JSON object containing the following fields:

```json
{
    "jsonrpc": "2.0",
    "method": "mev_cancelBundle",
    "params": [
        {
            "id": "0x1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd" // bundle id to cancel
        }
    ],
    "id": 1
}
```

Successful responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": null
}
```

Error responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {
        "code": -3200, // error code,
        "message": "error message", // error message
    }
}
```

### `mev_sendPrivateTransaction`

This method is used to send a private transaction to the builder service. The request body should be a JSON object containing the following fields:

```json
{
    "jsonrpc": "2.0",
    "method": "mev_sendPrivateTransaction",
    "params": [
        [
            {
                "tx": "0x0000...d46e8dd67c5d32be8d46675058bb8eb970870f072445675" // hex-encoded signed transaction bytes
            }
        ]
    ],
    "id": 1
}
```

Successful responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        { 
            "hash": "0x123...", // transaction hash
            "error": "" // error message if the transaction had an error and could not be added
        }
    ]
}
```

Error responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {
        "code": -3200, // error code,
        "message": "error message", // error message
    }
}
```

### `mev_sendPrivateRawTransaction`

This method is used to send a private transaction to the builder service. The request body should be a JSON object containing the following fields:

```json
{
    "jsonrpc": "2.0",
    "method": "mev_sendPrivateRawTransaction",
    "params": [
        [ 
        "0x0000...d46e8dd67c5d32be8d46675058bb8eb970870f072445675" // hex-encoded signed transaction bytes
        ]
    ],
    "id": 1
}
```

Successful responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        { 
            "hash": "0x123...", // transaction hash
            "error": "" // error message if the transaction had an error and could not be added
        }
    ]
}
```

Error responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {
        "code": -3200, // error code,
        "message": "error message", // error message
    }
}
```

### `mev_cancelPrivateTransaction`

This method is used to cancel a private transaction to the builder service. The request body should be a JSON object containing the following fields:

```json
{
    "jsonrpc": "2.0",
    "method": "mev_cancelPrivateTransaction",
    "params": [
        [
            {
                "txHash": "0xf0b..." // tx hash of the transaction to be canceled
            }
        ]
    ],
    "id": 1
}
```

Successful responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
      "0x123...", // transaction hash
    ]
}
```

Error responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {
        "code": -3200, // error code,
        "message": "error message", // error message
    }
}
```

### `mev_bundleServiceStatus`

This method is used to get the status of the bundle service. The request body should be a JSON object containing the following fields:

```json
{
    "jsonrpc": "2.0",
    "method": "mev_bundleServiceStatus",
    "params": [],
    "id": 1
}
```

Successful responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "enabled" // bundle service status
}
```

Error responses:

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {
        "code": -3200, // error code,
        "message": "error message", // error message
    }
}
```

**Important notice about bundles:** Bundles are processed using the following priority rules:

* **Block Number:** Earliest block numbers come first.
* **Max Payout:** Higher total gas values are prioritized.
* **Timestamp:** Earlier received timestamps are processed earlier.
* **Max Timestamp:** Bundles with the earliest max timestamp are processed first.
* **Reverting Transactions:** Fewer reverting transactions lead to higher priority.
* **Total Transactions:** Bundles with fewer transactions are processed sooner.

The algorithm proceeds to the next priority only if the current priority is a tie.

### Other Geth RPC Calls

The Builder service also exposes all of the standard Geth RPC calls.

The full list of Geth RPC calls can be found [here](https://geth.ethereum.org/docs/rpc/server). PoN Builder supports all standard [JSON-RPC-API](https://github.com/ethereum/execution-apis) endpoints.

## `geth` specific flags

### Full node on the main Ethereum network

By far the most common scenario is people wanting to simply interact with the Ethereum
network: create accounts; transfer funds; deploy and interact with contracts. For this
particular use case, the user doesn't care about years-old historical data, so we can
sync quickly to the current state of the network. To do so:

```shell
geth console
```

This command will:

* Start `geth` in snap sync mode (default, can be changed with the `--syncmode` flag),
   causing it to download more data in exchange for avoiding processing the entire history
   of the Ethereum network, which is very CPU intensive.
* Start the built-in interactive [JavaScript console](https://geth.ethereum.org/docs/interacting-with-geth/javascript-console),
   (via the trailing `console` subcommand) through which you can interact using [`web3` methods](https://github.com/ChainSafe/web3.js/blob/0.20.7/DOCUMENTATION.md)
   (note: the `web3` version bundled within `geth` is very old, and not up to date with official docs),
   as well as `geth`'s own [management APIs](https://geth.ethereum.org/docs/interacting-with-geth/rpc).
   This tool is optional and if you leave it out you can always attach it to an already running
   `geth` instance with `geth attach`.

### A Full node on the Görli test network

Transitioning towards developers, if you'd like to play around with creating Ethereum
contracts, you almost certainly would like to do that without any real money involved until
you get the hang of the entire system. In other words, instead of attaching to the main
network, you want to join the **test** network with your node, which is fully equivalent to
the main network, but with play-Ether only.

```shell
geth --goerli console
```

The `console` subcommand has the same meaning as above and is equally
useful on the testnet too.

Specifying the `--goerli` flag, however, will reconfigure your `geth` instance a bit:

* Instead of connecting to the main Ethereum network, the client will connect to the Görli
   test network, which uses different P2P bootnodes, different network IDs and genesis
   states.
* Instead of using the default data directory (`~/.ethereum` on Linux for example), `geth`
   will nest itself one level deeper into a `goerli` subfolder (`~/.ethereum/goerli` on
   Linux). Note, on OSX and Linux this also means that attaching to a running testnet node
   requires the use of a custom endpoint since `geth attach` will try to attach to a
   production node endpoint by default, e.g.,
   `geth attach <datadir>/goerli/geth.ipc`. Windows users are not affected by
   this.

*Note: Although some internal protective measures prevent transactions from
crossing over between the main network and test network, you should always
use separate accounts for play and real money. Unless you manually move
accounts, `geth` will by default correctly separate the two networks and will not make any
accounts available between them.*

### Configuration

As an alternative to passing the numerous flags to the `geth` binary, you can also pass a
configuration file via:

```shell
geth --config /path/to/your_config.toml
```

To get an idea of how the file should look like you can use the `dumpconfig` subcommand to
export your existing configuration:

```shell
geth --your-favourite-flags dumpconfig
```

*Note: This works only with `geth` v1.6.0 and above.*

#### Docker quick start

One of the quickest ways to get Ethereum up and running on your machine is by using
Docker:

```shell
docker run -d --name ethereum-node -v /Users/alice/ethereum:/root \
           -p 8545:8545 -p 30303:30303 \
           ethereum/client-go
```

This will start `geth` in snap-sync mode with a DB memory allowance of 1GB, as the
above command does.  It will also create a persistent volume in your home directory for
saving your blockchain as well as map the default ports. There is also an `alpine` tag
available for a slim version of the image.

Do not forget `--http.addr 0.0.0.0`, if you want to access RPC from other containers
and/or hosts. By default, `geth` binds to the local interface and RPC endpoints are not
accessible from the outside.

### Programmatically interfacing `geth` nodes

As a developer, sooner rather than later you'll want to start interacting with `geth` and the
Ethereum network via your own programs and not manually through the console. To aid
this, `geth` has built-in support for a JSON-RPC based APIs ([standard APIs](https://ethereum.github.io/execution-apis/api-documentation/)
and [`geth` specific APIs](https://geth.ethereum.org/docs/interacting-with-geth/rpc)).
These can be exposed via HTTP, WebSockets and IPC (UNIX sockets on UNIX based
platforms, and named pipes on Windows).

The IPC interface is enabled by default and exposes all the APIs supported by `geth`,
whereas the HTTP and WS interfaces need to manually be enabled and only expose a
subset of APIs due to security reasons. These can be turned on/off and configured as
you'd expect.

HTTP based JSON-RPC API options:

* `--http` Enable the HTTP-RPC server
* `--http.addr` HTTP-RPC server listening interface (default: `localhost`)
* `--http.port` HTTP-RPC server listening port (default: `8545`)
* `--http.api` API's offered over the HTTP-RPC interface (default: `eth,net,web3`)
* `--http.corsdomain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
* `--ws` Enable the WS-RPC server
* `--ws.addr` WS-RPC server listening interface (default: `localhost`)
* `--ws.port` WS-RPC server listening port (default: `8546`)
* `--ws.api` API's offered over the WS-RPC interface (default: `eth,net,web3`)
* `--ws.origins` Origins from which to accept WebSocket requests
* `--ipcdisable` Disable the IPC-RPC server
* `--ipcapi` API's offered over the IPC-RPC interface (default: `admin,debug,eth,miner,net,personal,txpool,web3`)
* `--ipcpath` Filename for IPC socket/pipe within the datadir (explicit paths escape it)

You'll need to use your own programming environments' capabilities (libraries, tools, etc) to
connect via HTTP, WS or IPC to a `geth` node configured with the above flags and you'll
need to speak [JSON-RPC](https://www.jsonrpc.org/specification) on all transports. You
can reuse the same connection for multiple requests!

**Note: Please understand the security implications of opening up an HTTP/WS based
transport before doing so! Hackers on the internet are actively trying to subvert
Ethereum nodes with exposed APIs! Further, all browser tabs can access locally
running web servers, so malicious web pages could try to subvert locally available
APIs!**

### Operating a private network

Maintaining your own private network is more involved as a lot of configurations taken for
granted in the official networks need to be manually set up.

#### Defining the private genesis state

First, you'll need to create the genesis state of your networks, which all nodes need to be
aware of and agree upon. This consists of a small JSON file (e.g. call it `genesis.json`):

```json
{
  "config": {
    "chainId": <arbitrary positive integer>,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "berlinBlock": 0,
    "londonBlock": 0
  },
  "alloc": {},
  "coinbase": "0x0000000000000000000000000000000000000000",
  "difficulty": "0x20000",
  "extraData": "",
  "gasLimit": "0x2fefd8",
  "nonce": "0x0000000000000042",
  "mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp": "0x00"
}
```

The above fields should be fine for most purposes, although we'd recommend changing
the `nonce` to some random value so you prevent unknown remote nodes from being able
to connect to you. If you'd like to pre-fund some accounts for easier testing, create
the accounts and populate the `alloc` field with their addresses.

```json
"alloc": {
  "0x0000000000000000000000000000000000000001": {
    "balance": "111111111"
  },
  "0x0000000000000000000000000000000000000002": {
    "balance": "222222222"
  }
}
```

With the genesis state defined in the above JSON file, you'll need to initialize **every**
`geth` node with it prior to starting it up to ensure all blockchain parameters are correctly
set:

```shell
geth init path/to/genesis.json
```

#### Creating the rendezvous point

With all nodes that you want to run initialized to the desired genesis state, you'll need to
start a bootstrap node that others can use to find each other in your network and/or over
the internet. The clean way is to configure and run a dedicated bootnode:

```shell
bootnode --genkey=boot.key
bootnode --nodekey=boot.key
```

With the bootnode online, it will display an [`enode` URL](https://ethereum.org/en/developers/docs/networking-layer/network-addresses/#enode)
that other nodes can use to connect to it and exchange peer information. Make sure to
replace the displayed IP address information (most probably `[::]`) with your externally
accessible IP to get the actual `enode` URL.

*Note: You could also use a full-fledged `geth` node as a bootnode, but it's the less
recommended way.*

#### Starting up your member nodes

With the bootnode operational and externally reachable (you can try
`telnet <ip> <port>` to ensure it's indeed reachable), start every subsequent `geth`
node pointed to the bootnode for peer discovery via the `--bootnodes` flag. It will
probably also be desirable to keep the data directory of your private network separated, so
do also specify a custom `--datadir` flag.

```shell
geth --datadir=path/to/custom/data/folder --bootnodes=<bootnode-enode-url-from-above>
```

*Note: Since your network will be completely cut off from the main and test networks, you'll
also need to configure a miner to process transactions and create new blocks for you.*

#### Running a private miner

In a private network setting a single CPU miner instance is more than enough for
practical purposes as it can produce a stable stream of blocks at the correct intervals
without needing heavy resources (consider running on a single thread, no need for multiple
ones either). To start a `geth` instance for mining, run it with all your usual flags, extended
by:

```shell
geth <usual-flags> --mine --miner.threads=1 --miner.etherbase=0x0000000000000000000000000000000000000000
```

Which will start mining blocks and transactions on a single CPU thread, crediting all
proceedings to the account specified by `--miner.etherbase`. You can further tune the mining
by changing the default gas limit blocks converge to (`--miner.targetgaslimit`) and the price
transactions are accepted at (`--miner.gasprice`).

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) are licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.

The builder functionality (under the `/builder/` and `/miner/` directory) is licensed under MIT license defined in the following [file](./License).
