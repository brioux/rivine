package blockcreator

import (
	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/types"
)

// submitBlock accepts a block.
func (m *BlockCreator) submitBlock() error {
	if err := m.tg.Add(); err != nil {
		return err
	}
	defer m.tg.Done()

	// The first part needs to be wrapped in an anonymous function
	// for lock safety.
	b := types.Block{}
	err := func() error {
		m.mu.Lock()
		defer m.mu.Unlock()

		// Block is going to be passed to external memory, but the memory pointed
		// to by the transactions slice is still being modified - needs to be
		// copied. Same with the memory being pointed to by the arb data slice.
		txns := make([]types.Transaction, len(m.unsolvedBlock.Transactions))
		copy(txns, m.unsolvedBlock.Transactions)
		b.Transactions = txns

		return nil
	}()

	// Give the block to the consensus set.
	err = m.cs.AcceptBlock(b)

	if err == modules.ErrNonExtendingBlock {
		m.log.Println("Created a stale block - block appears valid but does not extend the blockchain")
		return err
	}
	if err == modules.ErrBlockUnsolved {
		m.log.Println("Created an unsolved block - submission appears to be incorrect")
		return err
	}
	if err != nil {
		m.tpool.PurgeTransactionPool()
		m.log.Critical("ERROR: an invalid block was submitted:", err)
		return err
	}
	return nil
}
