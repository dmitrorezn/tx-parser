package memory

import (
	"sync"
	"sync/atomic"
)

type BlockNumberStorage struct {
	currentBlock atomic.Int64

	mu                    sync.RWMutex
	processedTransactions map[int]int
}

func NewBlockNumberStorage() *BlockNumberStorage {
	return &BlockNumberStorage{
		processedTransactions: make(map[int]int),
	}
}

func (bs *BlockNumberStorage) GetCurrentBlock() int {
	return int(bs.currentBlock.Load())
}

func (bs *BlockNumberStorage) SetCurrentBlock(currBlock int) {
	bs.currentBlock.Store(int64(currBlock))
}

func (bs *BlockNumberStorage) DelLastProcessedTxIndex(blockNumber int) {
	bs.mu.Lock()
	delete(bs.processedTransactions, blockNumber)
	bs.mu.Unlock()
}

func (bs *BlockNumberStorage) GetLastProcessedTxIndex(block int) (int, bool) {
	bs.mu.Lock()
	idx, ok := bs.processedTransactions[block]
	bs.mu.Unlock()

	return idx, ok
}

func (bs *BlockNumberStorage) SetLastProcessedTxIndex(block int, idx int) {
	bs.mu.Lock()
	bs.processedTransactions[block] = idx
	bs.mu.Unlock()
}
