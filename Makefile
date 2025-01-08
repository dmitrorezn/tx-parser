run:
	go run ./cmd/server -addr localhost:8080 -blockStart 21508148 -interval 1s -eth_addr https://ethereum-rpc.publicnode.com

build:
	go build -o build/server ./cmd/server

test:
	go test ./... && go clean -testcache

exec:
	./build/server

getCurrBlock:
	curl -X GET http://localhost:8080/current-block

sub:
	curl -X POST http://localhost:8080/subscribe -d '{"address": "${ADDR}"}'

getTxs:
	curl -X GET http://localhost:8080/transactions/${ADDR}