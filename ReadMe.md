
# Transaction JSON RPC Proxy Server

This JSON RPC server acts as a proxy between the user and the Ethereum execution node. It enables transaction cancellation without the need to submit a transaction with a higher gas price to the mempool, thereby avoiding unnecessary loss of ETH. 

The server works by storing the transaction in memory and monitoring gas prices. When the gas price decreases sufficiently, the transaction's chance of inclusion in the next block increases.

## Available Methods

- `eth_sendRawTransaction`: This method is intercepted by the server which then stores the transaction until the chances of successful execution are significantly high. Additionally, this method plays a crucial role in cancelling transactions. When the server receives a transaction bearing the same nonce and value, intended for the server's wallet and accompanied by a higher gas price, it interprets this as a cancellation request. In both scenarios, the server mimics the behavior of a standard node by returning the transaction hash, thereby maintaining compatibility with MetaMask.

- `cancel_transaction`: This is a custom JSON RPC method implemented in the server. It deletes a transaction if it's in the "STORED" state and hasn't been submitted yet.

**Note:** All other RPC calls will be forwarded to the Ethereum Node.

## Setup

Update your `.env` file with your `INFURA_PROJECT_ID`:

```
INFURA_PROJECT_ID=<YOUR_PROJECT_ID>
NETWORK=goerli
HOST=0.0.0.0
PORT=8080
LOG_LEVEL=INFO
```
Additional configuration options are available in this file.

### How to Run

You can build and run the project with this command:

```
go build . && ./json-rpc-proxy
```

## Testing

To run unit tests, use the following command:

```
go test ./... -cover
```

Automated tests using ethers.js are located in the 'test' directory. You can configure your .env file for these tests:

```
SERVER_ADDRESS=http://localhost:8080
PRIVATE_KEY=WALLET_PRIVATE_KEY
TO_ADDRESS=RECIPIENT_ADDRESS
```


