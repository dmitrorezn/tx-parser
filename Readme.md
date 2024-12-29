# Tx Parser - Ethereum Blockchain Parser

## Goal

The **Tx Parser** is a Go-based implementation designed to parse Ethereum blockchain transactions and query inbound/outbound transactions for subscribed addresses. It is intended to integrate seamlessly with a notification service to provide real-time updates for user-specified Ethereum addresses.

---

## Features

- **Subscription-based Monitoring**: Add Ethereum addresses to a subscription list to monitor transactions.
- **Transaction Querying**: Fetch inbound and outbound transactions for any subscribed address.
- **Memory Storage**: Efficient in-memory storage with a modular design to support future storage implementations.
- **Block Tracking**: Track the last parsed Ethereum block to ensure consistent parsing.
- **HTTP API**: Expose functionality for easy external usage.

---

## Prerequisites

To build and run the Tx Parser, you need:

- [Go](https://golang.org/dl/) (version 1.22 or later)
- An Ethereum node with JSON-RPC enabled  address

---

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/dmitrorezn/tx-parser.git
   cd tx-parser
   
Build the binary:

```bash
go build -o server
```
Run the application:

```bash
./server
```
Usage
HTTP Interface:
```bash
ADDR - env varable with address to subscribe or get transactions 
```
for usage replace 0x00 in variable ADDR  with your address

Subscribe:
```bash
	ADDR=0x00 curl -X POST http://localhost:8080/subscribe -d '{"address": "${ADDR}"}'
```
Get Transactions:
```bash
	ADDR=0x00 curl -X GET http://localhost:8080/transactions/${ADDR}```
```
Get Current Block:
```bash
	ADDR=0x00 curl -X GET http://localhost:8080/current-block
```

HTTP API

Available Endpoints:
```
Method	   Endpoint	            Description
POST       /subscribe	            Add an Ethereum address to the observer list
GET	   /transactions/{address}	Fetch inbound/outbound transactions for address
GET	   /current-block	        Get the last parsed Ethereum block
```
Implementation Details

The Service interface ensures a consistent API for parsing and querying Ethereum transactions:

```go
type Parser interface {
    // Get the last parsed Ethereum block
    GetCurrentBlock() int

    // Add an address to the subscription list
    Subscribe(address string) bool

    // Get transactions for a subscribed address
    GetTransactions(address string) []Transaction
}
```
Memory Storage
The initial implementation uses an in-memory storage system for:

Subscribed addresses.
Transactions for each subscribed address.
Last parsed block.
This can be extended to external storage solutions such as Redis, PostgreSQL, or others by abstracting storage operations.

Ethereum JSON-RPC Integration
To interact with the Ethereum blockchain, the Tx Parser uses JSON-RPC:


Run Tests:

```bash
make run
```

Run Tests:

```bash
make tests
```
Get Curr Block by curl HTTP API call:
```bash
make getCurrBlock
```
Subscribe by curl HTTP API call:
```bash
make sub ADDR=0x00
```
Get Address transactions by curl HTTP API call:
```bash
getTxs ADDR=0x00
```

Logging and Debugging: Uses slog JSON Handler to define structured logs in system
