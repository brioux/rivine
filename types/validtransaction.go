package types

// validtransaction.go has functions for checking whether a transaction is
// valid outside of the context of a consensus set. This means checking the
// size of the transaction, the content of the signatures, and a large set of
// other rules that are inherent to how a transaction should be constructed.

import (
	"errors"

	"github.com/rivine/rivine/encoding"
)

var (
	ErrDoubleSpend          = errors.New("transaction uses a parent object twice")
	ErrNonZeroRevision      = errors.New("new file contract has a nonzero revision number")
	ErrTimelockNotSatisfied = errors.New("timelock has not been met")
	ErrTransactionTooLarge  = errors.New("transaction is too large to fit in a block")
	ErrZeroMinerFee         = errors.New("transaction has a zero value miner fee")
	ErrZeroOutput           = errors.New("transaction cannot have an output or payout that has zero value")
)

// fitsInABlock checks if the transaction is likely to fit in a block.
// Currently there is no limitation on transaction size other than it must fit
// in a block.
func (t Transaction) fitsInABlock() error {
	// Check that the transaction will fit inside of a block, leaving 5kb for
	// overhead.
	if uint64(len(encoding.Marshal(t))) > BlockSizeLimit-5e3 {
		return ErrTransactionTooLarge
	}
	return nil
}

// followsMinimumValues checks that all outputs adhere to the rules for the
// minimum allowed value (generally 1).
func (t Transaction) followsMinimumValues() error {
	for _, sco := range t.CoinOutputs {
		if sco.Value.IsZero() {
			return ErrZeroOutput
		}
	}
	for _, bso := range t.BlockStakeOutputs {
		if bso.Value.IsZero() {
			return ErrZeroOutput
		}
	}
	for _, fee := range t.MinerFees {
		if fee.IsZero() {
			return ErrZeroMinerFee
		}
	}
	return nil
}

// noRepeats checks that a transaction does not spend multiple outputs twice,
// submit two valid storage proofs for the same file contract, etc. We
// frivilously check that a file contract termination and storage proof don't
// act on the same file contract. There is very little overhead for doing so,
// and the check is only frivilous because of the current rule that file
// contract terminations are not valid after the proof window opens.
func (t Transaction) noRepeats() error {
	// Check that there are no repeat instances of coin outputs, storage
	// proofs, contract terminations, or siafund outputs.
	coinInputs := make(map[CoinOutputID]struct{})
	for _, sci := range t.CoinInputs {
		_, exists := coinInputs[sci.ParentID]
		if exists {
			return ErrDoubleSpend
		}
		coinInputs[sci.ParentID] = struct{}{}
	}
	blockstakeInputs := make(map[BlockStakeOutputID]struct{})
	for _, bsi := range t.BlockStakeInputs {
		_, exists := blockstakeInputs[bsi.ParentID]
		if exists {
			return ErrDoubleSpend
		}
		blockstakeInputs[bsi.ParentID] = struct{}{}
	}
	return nil
}

// validUnlockConditions checks that the conditions of uc have been met. The
// height is taken as input so that modules who might be at a different height
// can do the verification without needing to use their own function.
// Additionally, it means that the function does not need to be a method of the
// consensus set.
func validUnlockConditions(uc UnlockConditions, currentHeight BlockHeight) (err error) {
	if uc.Timelock > currentHeight {
		return ErrTimelockNotSatisfied
	}
	return
}

// validUnlockConditions checks that all of the unlock conditions in the
// transaction are valid.
func (t Transaction) validUnlockConditions(currentHeight BlockHeight) (err error) {
	for _, sci := range t.CoinInputs {
		err = validUnlockConditions(sci.UnlockConditions, currentHeight)
		if err != nil {
			return
		}
	}
	for _, bsi := range t.BlockStakeInputs {
		err = validUnlockConditions(bsi.UnlockConditions, currentHeight)
		if err != nil {
			return
		}
	}
	return
}

// StandaloneValid returns an error if a transaction is not valid in any
// context, for example if the same output is spent twice in the same
// transaction. StandaloneValid will not check that all outputs being spent are
// legal outputs, as it has no confirmed or unconfirmed set to look at.
func (t Transaction) StandaloneValid(currentHeight BlockHeight) (err error) {
	err = t.fitsInABlock()
	if err != nil {
		return
	}
	err = t.noRepeats()
	if err != nil {
		return
	}
	err = t.followsMinimumValues()
	if err != nil {
		return
	}
	err = t.validUnlockConditions(currentHeight)
	if err != nil {
		return
	}
	err = t.validSignatures(currentHeight)
	if err != nil {
		return
	}
	return
}
