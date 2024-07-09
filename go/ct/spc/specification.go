// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package spc

//go:generate mockgen -source specification.go -destination specification_mock.go -package spc

import (
	"fmt"
	"slices"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	. "github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// Specification defines the interface for handling specifications.
type Specification interface {
	// GetRules provides access to all rules within the specification.
	GetRules() []Rule

	// GetRulesFor gives you all rules that apply to the given State (i.e. where
	// the rule's Condition holds).
	GetRulesFor(*st.State) []Rule
}

var Spec = func() Specification {
	return NewSpecificationMap(getAllRules()...)
}()

// instruction holds the basic information for the 4 basic rules
// these are not enough gas, stack overflow, stack underflow, and a regular behavior case
type instruction struct {
	op         tosca.OpCode
	staticGas  tosca.Gas
	pops       int
	pushes     int
	conditions []Condition       // conditions for the regular case
	parameters []Parameter       // parameters for the regular case
	effect     func(s *st.State) // effect for the regular case
	name       string
}

////////////////////////////////////////////////////////////

func boolToU256(value bool) U256 {
	if value {
		return NewU256(1)
	}
	return NewU256(0)
}

func getAllRules() []Rule {
	rules := []Rule{}

	// --- Terminal States ---

	rules = append(rules, []Rule{
		{
			Name:      "stopped_is_end",
			Condition: And(AnyKnownRevision(), Eq(Status(), st.Stopped)),
			Effect:    NoEffect(),
		},

		{
			Name:      "reverted_is_end",
			Condition: And(AnyKnownRevision(), Eq(Status(), st.Reverted)),
			Effect:    NoEffect(),
		},

		{
			Name:      "failed_is_end",
			Condition: And(AnyKnownRevision(), Eq(Status(), st.Failed)),
			Effect:    NoEffect(),
		},

		{
			Name:      "pc_on_data_is_ignored",
			Condition: IsData(Pc()),
			Effect:    NoEffect(),
		},
	}...)

	// --- Invalid Instructions ---

	for i := 0; i < 256; i++ {
		op := tosca.OpCode(i)
		if !tosca.IsValid(op) {
			rules = append(rules, Rule{
				Name:      fmt.Sprintf("%v_invalid", op),
				Condition: And(Eq(Status(), st.Running), Eq(Op(Pc()), op)),
				Effect:    FailEffect(),
			})
		}
	}

	// --- Error States ---

	rules = append(rules, []Rule{
		{
			Name:      "unknown_revision_is_end",
			Condition: IsRevision(R99_UnknownNextRevision),
			Effect:    FailEffect(),
		},
	}...)

	// --- STOP ---

	rules = append(rules, Rule{
		Name: "stop_terminates_interpreter",
		Condition: And(
			AnyKnownRevision(),
			Eq(Status(), st.Running),
			Eq(Op(Pc()), tosca.STOP),
		),
		Effect: Change(func(s *st.State) {
			s.Status = st.Stopped
			s.ReturnData = Bytes{}
			s.Pc++
		}),
	})

	// --- Arithmetic ---

	rules = append(rules, binaryOp(tosca.ADD, 3, func(a, b U256) U256 {
		return a.Add(b)
	})...)

	rules = append(rules, binaryOp(tosca.MUL, 5, func(a, b U256) U256 {
		return a.Mul(b)
	})...)

	rules = append(rules, binaryOp(tosca.SUB, 3, func(a, b U256) U256 {
		return a.Sub(b)
	})...)

	rules = append(rules, binaryOp(tosca.DIV, 5, func(a, b U256) U256 {
		return a.Div(b)
	})...)

	rules = append(rules, binaryOp(tosca.SDIV, 5, func(a, b U256) U256 {
		return a.SDiv(b)
	})...)

	rules = append(rules, binaryOp(tosca.MOD, 5, func(a, b U256) U256 {
		return a.Mod(b)
	})...)

	rules = append(rules, binaryOp(tosca.SMOD, 5, func(a, b U256) U256 {
		return a.SMod(b)
	})...)

	rules = append(rules, trinaryOp(tosca.ADDMOD, 8, func(a, b, n U256) U256 {
		return a.AddMod(b, n)
	})...)

	rules = append(rules, trinaryOp(tosca.MULMOD, 8, func(a, b, n U256) U256 {
		return a.MulMod(b, n)
	})...)

	rules = append(rules, binaryOpWithDynamicCost(tosca.EXP, 10, func(a, e U256) U256 {
		return a.Exp(e)
	}, func(a, e U256) tosca.Gas {
		const gasFactor = tosca.Gas(50)
		expBytes := e.Bytes32be()
		for i := 0; i < 32; i++ {
			if expBytes[i] != 0 {
				return gasFactor * tosca.Gas(32-i)
			}
		}
		return 0
	})...)

	rules = append(rules, binaryOp(tosca.SIGNEXTEND, 5, func(b, x U256) U256 {
		return x.SignExtend(b)
	})...)

	rules = append(rules, binaryOp(tosca.LT, 3, func(a, b U256) U256 {
		return boolToU256(a.Lt(b))
	})...)

	rules = append(rules, binaryOp(tosca.GT, 3, func(a, b U256) U256 {
		return boolToU256(a.Gt(b))
	})...)

	rules = append(rules, binaryOp(tosca.SLT, 3, func(a, b U256) U256 {
		return boolToU256(a.Slt(b))
	})...)

	rules = append(rules, binaryOp(tosca.SGT, 3, func(a, b U256) U256 {
		return boolToU256(a.Sgt(b))
	})...)

	rules = append(rules, binaryOp(tosca.EQ, 3, func(a, b U256) U256 {
		return boolToU256(a.Eq(b))
	})...)

	rules = append(rules, unaryOp(tosca.ISZERO, 3, func(a U256) U256 {
		return boolToU256(a.IsZero())
	})...)

	rules = append(rules, binaryOp(tosca.AND, 3, func(a, b U256) U256 {
		return a.And(b)
	})...)

	rules = append(rules, binaryOp(tosca.OR, 3, func(a, b U256) U256 {
		return a.Or(b)
	})...)

	rules = append(rules, binaryOp(tosca.XOR, 3, func(a, b U256) U256 {
		return a.Xor(b)
	})...)

	rules = append(rules, unaryOp(tosca.NOT, 3, func(a U256) U256 {
		return a.Not()
	})...)

	rules = append(rules, binaryOp(tosca.BYTE, 3, func(i, x U256) U256 {
		if i.Gt(NewU256(31)) {
			return NewU256(0)
		}
		return NewU256(uint64(x.Bytes32be()[i.Uint64()]))
	})...)

	rules = append(rules, binaryOp(tosca.SHL, 3, func(shift, value U256) U256 {
		return value.Shl(shift)
	})...)

	rules = append(rules, binaryOp(tosca.SHR, 3, func(shift, value U256) U256 {
		return value.Shr(shift)
	})...)

	rules = append(rules, binaryOp(tosca.SAR, 3, func(shift, value U256) U256 {
		return value.Srsh(shift)
	})...)

	// --- SHA3 ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.SHA3,
		staticGas: 30,
		pops:      2,
		pushes:    1,
		parameters: []Parameter{
			MemoryOffsetParameter{},
			MemorySizeParameter{},
		},
		effect: func(s *st.State) {
			offsetU256 := s.Stack.Pop()
			sizeU256 := s.Stack.Pop()

			memExpCost, offset, size := s.Memory.ExpansionCosts(offsetU256, sizeU256)
			if s.Gas < memExpCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= memExpCost

			wordCost := tosca.Gas(6 * SizeInWords(size))
			if s.Gas < wordCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= wordCost

			hash := s.Memory.Hash(offset, size)
			s.Stack.Push(NewU256FromBytes(hash[:]...))
		},
	})...)

	// --- BALANCE ---

	// cold
	rules = append(rules, rulesFor(instruction{
		op:        tosca.BALANCE,
		staticGas: 0 + 2600, // 2600 dynamic cost for cold address
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsAddressCold(Param(0)),
		},
		parameters: []Parameter{
			AddressParameter{},
		},
		effect: func(s *st.State) {
			address := NewAddress(s.Stack.Pop())
			s.Stack.Push(s.Accounts.GetBalance(address))
			s.Accounts.MarkWarm(address)
		},
		name: "_cold",
	})...)

	// warm
	rules = append(rules, rulesFor(instruction{
		op:        tosca.BALANCE,
		staticGas: 0 + 100, // 100 dynamic cost for warm address
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsAddressWarm(Param(0)),
		},
		parameters: []Parameter{
			AddressParameter{},
		},
		effect: func(s *st.State) {
			address := NewAddress(s.Stack.Pop())
			s.Stack.Push(s.Accounts.GetBalance(address))
		},
		name: "_warm",
	})...)

	// pre Berlin
	rules = append(rules, rulesFor(instruction{
		op:        tosca.BALANCE,
		staticGas: 700,
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			IsRevision(tosca.R07_Istanbul),
		},
		parameters: []Parameter{
			AddressParameter{},
		},
		effect: func(s *st.State) {
			address := NewAddress(s.Stack.Pop())
			s.Stack.Push(s.Accounts.GetBalance(address))
		},
		name: "_preBerlin",
	})...)

	// --- MLOAD ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.MLOAD,
		staticGas: 3,
		pops:      1,
		pushes:    1,
		parameters: []Parameter{
			MemoryOffsetForCopyParameter{},
		},
		effect: func(s *st.State) {
			offsetU256 := s.Stack.Pop()

			cost, offset, _ := s.Memory.ExpansionCosts(offsetU256, NewU256(32))
			if s.Gas < cost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= cost

			value := NewU256FromBytes(s.Memory.Read(offset, 32)...)
			s.Stack.Push(value)
		},
	})...)

	// --- MSTORE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.MSTORE,
		staticGas: 3,
		pops:      2,
		pushes:    0,
		parameters: []Parameter{
			MemoryOffsetForCopyParameter{},
			NumericParameter{},
		},
		effect: func(s *st.State) {
			offsetU256 := s.Stack.Pop()
			value := s.Stack.Pop()

			cost, offset, _ := s.Memory.ExpansionCosts(offsetU256, NewU256(32))
			if s.Gas < cost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= cost

			bytes := value.Bytes32be()
			s.Memory.Write(bytes[:], offset)
		},
	})...)

	// --- MSTORE8 ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.MSTORE8,
		staticGas: 3,
		pops:      2,
		pushes:    0,
		parameters: []Parameter{
			MemoryOffsetForCopyParameter{},
			NumericParameter{},
		},
		effect: func(s *st.State) {
			offsetU256 := s.Stack.Pop()
			value := s.Stack.Pop()

			cost, offset, _ := s.Memory.ExpansionCosts(offsetU256, NewU256(1))
			if s.Gas < cost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= cost

			s.Memory.Write([]byte{value.Bytes32be()[31]}, offset)

		},
	})...)

	// --- SLOAD ---

	// cold
	rules = append(rules, rulesFor(instruction{
		op:        tosca.SLOAD,
		staticGas: 100 + 2000, // 2000 are from the dynamic cost of cold mem
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsStorageCold(Param(0)),
		},
		parameters: []Parameter{
			NumericParameter{},
		},
		effect: func(s *st.State) {
			key := s.Stack.Pop()
			s.Stack.Push(s.Storage.GetCurrent(key))
			s.Storage.MarkWarm(key)
		},
		name: "_cold",
	})...)

	// warm
	rules = append(rules, rulesFor(instruction{
		op:        tosca.SLOAD,
		staticGas: 100,
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsStorageWarm(Param(0)),
		},
		parameters: []Parameter{
			NumericParameter{},
		},
		effect: func(s *st.State) {
			key := s.Stack.Pop()
			s.Stack.Push(s.Storage.GetCurrent(key))
		},
		name: "_warm",
	})...)

	// pre_berlin
	rules = append(rules, rulesFor(instruction{
		op:        tosca.SLOAD,
		staticGas: 800,
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			IsRevision(tosca.R07_Istanbul),
		},
		parameters: []Parameter{
			NumericParameter{},
		},
		effect: func(s *st.State) {
			key := s.Stack.Pop()
			s.Stack.Push(s.Storage.GetCurrent(key))
		},
		name: "_pre_berlin",
	})...)

	// --- SSTORE ---

	sstoreRules := []sstoreOpParams{
		{revision: tosca.R07_Istanbul, config: tosca.StorageAssigned, gasCost: 800},
		{revision: tosca.R07_Istanbul, config: tosca.StorageAdded, gasCost: 20000},
		{revision: tosca.R07_Istanbul, config: tosca.StorageAddedDeleted, gasCost: 800, gasRefund: 19200},
		{revision: tosca.R07_Istanbul, config: tosca.StorageDeletedRestored, gasCost: 800, gasRefund: -10800},
		{revision: tosca.R07_Istanbul, config: tosca.StorageDeletedAdded, gasCost: 800, gasRefund: -15000},
		{revision: tosca.R07_Istanbul, config: tosca.StorageDeleted, gasCost: 5000, gasRefund: 15000},
		{revision: tosca.R07_Istanbul, config: tosca.StorageModified, gasCost: 5000},
		{revision: tosca.R07_Istanbul, config: tosca.StorageModifiedDeleted, gasCost: 800, gasRefund: 15000},
		{revision: tosca.R07_Istanbul, config: tosca.StorageModifiedRestored, gasCost: 800, gasRefund: 4200},

		// Certain storage configurations imply warm access. Not all
		// combinations are possible; invalid ones are marked below.

		// {revision: tosca.R09_Berlin, warm: false, config: tosca.StorageAssigned, gasCost: 2200}, // invalid
		{revision: tosca.R09_Berlin, warm: false, config: tosca.StorageAdded, gasCost: 22100},
		// {revision: tosca.R09_Berlin, warm: false, config: tosca.StorageAddedDeleted, gasCost: 2200, gasRefund: 19900},     // invalid
		// {revision: tosca.R09_Berlin, warm: false, config: tosca.StorageDeletedRestored, gasCost: 2200, gasRefund: -10800}, // invalid
		// {revision: tosca.R09_Berlin, warm: false, config: tosca.StorageDeletedAdded, gasCost: 2200, gasRefund: -15000},    // invalid
		{revision: tosca.R09_Berlin, warm: false, config: tosca.StorageDeleted, gasCost: 5000, gasRefund: 15000},
		{revision: tosca.R09_Berlin, warm: false, config: tosca.StorageModified, gasCost: 5000},
		// {revision: tosca.R09_Berlin, warm: false, config: tosca.StorageModifiedDeleted, gasCost: 2200, gasRefund: 15000}, // invalid
		// {revision: tosca.R09_Berlin, warm: false, config: tosca.StorageModifiedRestored, gasCost: 2200, gasRefund: 4900}, // invalid

		{revision: tosca.R09_Berlin, warm: true, config: tosca.StorageAssigned, gasCost: 100},
		{revision: tosca.R09_Berlin, warm: true, config: tosca.StorageAdded, gasCost: 20000},
		{revision: tosca.R09_Berlin, warm: true, config: tosca.StorageAddedDeleted, gasCost: 100, gasRefund: 19900},
		{revision: tosca.R09_Berlin, warm: true, config: tosca.StorageDeletedRestored, gasCost: 100, gasRefund: -12200},
		{revision: tosca.R09_Berlin, warm: true, config: tosca.StorageDeletedAdded, gasCost: 100, gasRefund: -15000},
		{revision: tosca.R09_Berlin, warm: true, config: tosca.StorageDeleted, gasCost: 2900, gasRefund: 15000},
		{revision: tosca.R09_Berlin, warm: true, config: tosca.StorageModified, gasCost: 2900},
		{revision: tosca.R09_Berlin, warm: true, config: tosca.StorageModifiedDeleted, gasCost: 100, gasRefund: 15000},
		{revision: tosca.R09_Berlin, warm: true, config: tosca.StorageModifiedRestored, gasCost: 100, gasRefund: 2800},
	}

	for rev := tosca.R10_London; rev <= NewestSupportedRevision; rev++ {
		// Certain storage configurations imply warm access. Not all
		// combinations are possible; invalid ones are marked below.
		sstoreRules = append(sstoreRules, []sstoreOpParams{
			// {revision: rev, warm: false, config: tosca.StorageAssigned, gasCost: 2200}, // invalid
			{revision: rev, warm: false, config: tosca.StorageAdded, gasCost: 22100},
			// {revision: rev, warm: false, config: tosca.StorageAddedDeleted, gasCost: 2200, gasRefund: 19900},  // invalid
			// {revision: rev, warm: false, config: tosca.StorageDeletedRestored, gasCost: 2200, gasRefund: 100}, // invalid
			// {revision: rev, warm: false, config: tosca.StorageDeletedAdded, gasCost: 2200, gasRefund: -4800},  // invalid
			{revision: rev, warm: false, config: tosca.StorageDeleted, gasCost: 5000, gasRefund: 4800},
			{revision: rev, warm: false, config: tosca.StorageModified, gasCost: 5000},
			// {revision: rev, warm: false, config: tosca.StorageModifiedDeleted, gasCost: 2200, gasRefund: 4800},  // invalid
			// {revision: rev, warm: false, config: tosca.StorageModifiedRestored, gasCost: 2200, gasRefund: 4900}, // invalid

			{revision: rev, warm: true, config: tosca.StorageAssigned, gasCost: 100},
			{revision: rev, warm: true, config: tosca.StorageAdded, gasCost: 20000},
			{revision: rev, warm: true, config: tosca.StorageAddedDeleted, gasCost: 100, gasRefund: 19900},
			{revision: rev, warm: true, config: tosca.StorageDeletedRestored, gasCost: 100, gasRefund: -2000},
			{revision: rev, warm: true, config: tosca.StorageDeletedAdded, gasCost: 100, gasRefund: -4800},
			{revision: rev, warm: true, config: tosca.StorageDeleted, gasCost: 2900, gasRefund: 4800},
			{revision: rev, warm: true, config: tosca.StorageModified, gasCost: 2900},
			{revision: rev, warm: true, config: tosca.StorageModifiedDeleted, gasCost: 100, gasRefund: 4800},
			{revision: rev, warm: true, config: tosca.StorageModifiedRestored, gasCost: 100, gasRefund: 2800},
		}...)
	}

	for _, params := range sstoreRules {
		rules = append(rules, sstoreOpRegular(params))
		rules = append(rules, sstoreOpTooLittleGas(params))
		rules = append(rules, sstoreOpReadOnlyMode(params))
	}

	rules = append(rules, tooLittleGas(instruction{op: tosca.SSTORE, staticGas: 2300, name: "_EIP2200"})...)
	rules = append(rules, tooFewElements(instruction{op: tosca.SSTORE, staticGas: 2, pops: 2})...)

	// --- JUMP ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.JUMP,
		staticGas: 8,
		pops:      1,
		pushes:    0,
		parameters: []Parameter{
			JumpTargetParameter{},
		},
		conditions: []Condition{
			IsCode(Param(0)),
			Eq(Op(Param(0)), tosca.JUMPDEST),
		},
		effect: func(s *st.State) {
			target := s.Stack.Pop()
			s.Pc = uint16(target.Uint64())
		},
	})...)

	rules = append(rules, []Rule{
		{
			Name: "jump_to_data",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.JUMP),
				Ge(Gas(), 8),
				Ge(StackSize(), 1),
				IsData(Param(0)),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jump_to_invalid_destination",
			Parameter: []Parameter{
				JumpTargetParameter{},
			},
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.JUMP),
				Ge(Gas(), 8),
				Ge(StackSize(), 1),
				IsCode(Param(0)),
				Ne(Op(Param(0)), tosca.JUMPDEST),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- JUMPI ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.JUMPI,
		staticGas: 10,
		pops:      2,
		pushes:    0,
		parameters: []Parameter{
			JumpTargetParameter{},
			NumericParameter{},
		},
		conditions: []Condition{
			IsCode(Param(0)),
			Eq(Op(Param(0)), tosca.JUMPDEST),
			Ne(Param(1), NewU256(0)),
		},
		effect: func(s *st.State) {
			target := s.Stack.Pop()
			s.Stack.Pop()
			s.Pc = uint16(target.Uint64())
		},
	})...)

	rules = append(rules, []Rule{
		{
			Name: "jumpi_not_taken",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.JUMPI),
				Ge(Gas(), 10),
				Ge(StackSize(), 2),
				Eq(Param(1), NewU256(0)),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 10
				s.Stack.Pop()
				s.Stack.Pop()
				s.Pc += 1
			}),
		},

		{
			Name: "jumpi_to_data",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.JUMPI),
				Ge(Gas(), 10),
				Ge(StackSize(), 2),
				IsData(Param(0)),
				Ne(Param(1), NewU256(0)),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jumpi_to_invalid_destination",
			Parameter: []Parameter{
				JumpTargetParameter{},
				NumericParameter{},
			},
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.JUMPI),
				Ge(Gas(), 10),
				Ge(StackSize(), 2),
				IsCode(Param(0)),
				Ne(Op(Param(0)), tosca.JUMPDEST),
				Ne(Param(1), NewU256(0)),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- PC ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.PC,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(uint64(s.Pc) - 1))
		},
	})...)

	// --- MSIZE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.MSIZE,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(uint64(s.Memory.Size())))
		},
	})...)

	// --- GAS ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.GAS,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(uint64(s.Gas)))
		},
	})...)

	// --- JUMPDEST ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.JUMPDEST,
		staticGas: 1,
		pops:      0,
		pushes:    0,
		effect:    NoEffect().Apply,
	})...)

	// --- TLOAD ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.TLOAD,
		staticGas: 100,
		pops:      1,
		pushes:    1,
		parameters: []Parameter{
			StorageAccessKeyParameter{},
		},
		conditions: []Condition{
			RevisionBounds(tosca.R13_Cancun, NewestSupportedRevision),
			BindTransientStorageToNonZero(Param(0)),
		},
		effect: func(s *st.State) {
			key := s.Stack.Pop()
			value := s.TransientStorage.Get(key)
			s.Stack.Push(value)
		},
	})...)

	rules = append(rules, rulesFor(instruction{
		op:        tosca.TLOAD,
		staticGas: 100,
		pops:      1,
		pushes:    1,
		parameters: []Parameter{
			StorageAccessKeyParameter{},
		},
		conditions: []Condition{
			RevisionBounds(tosca.R13_Cancun, NewestSupportedRevision),
			BindTransientStorageToZero(Param(0)),
		},
		effect: func(s *st.State) {
			s.Stack.Pop()
			s.Stack.Push(NewU256(0))
		},
	})...)

	rules = append(rules, Rule{
		Name: "tload_pre_cancun",
		Condition: And(
			RevisionBounds(tosca.R07_Istanbul, tosca.R12_Shanghai),
			Eq(Status(), st.Running),
			Eq(Op(Pc()), tosca.TLOAD),
		),
		Effect: FailEffect(),
	})

	// --- TSTORE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.TSTORE,
		staticGas: 100,
		pops:      2,
		pushes:    0,
		parameters: []Parameter{
			StorageAccessKeyParameter{},
			NumericParameter{},
		},
		conditions: []Condition{
			RevisionBounds(tosca.R13_Cancun, NewestSupportedRevision),
			Eq(ReadOnly(), false),
			BindTransientStorageToNonZero(Param(0)),
		},
		effect: func(s *st.State) {
			key := s.Stack.Pop()
			value := s.Stack.Pop()
			s.TransientStorage.Set(key, value)
		},
	})...)

	rules = append(rules, rulesFor(instruction{
		op:        tosca.TSTORE,
		staticGas: 100,
		pops:      2,
		pushes:    0,
		parameters: []Parameter{
			StorageAccessKeyParameter{},
			NumericParameter{},
		},
		conditions: []Condition{
			RevisionBounds(tosca.R13_Cancun, NewestSupportedRevision),
			Eq(ReadOnly(), false),
			BindTransientStorageToZero(Param(0)),
		},
		effect: func(s *st.State) {
			key := s.Stack.Pop()
			value := s.Stack.Pop()
			s.TransientStorage.Set(key, value)
		},
	})...)

	rules = append(rules, Rule{
		Name: "tstore_pre_cancun",
		Condition: And(
			RevisionBounds(tosca.R07_Istanbul, tosca.R12_Shanghai),
			Eq(Status(), st.Running),
			Eq(Op(Pc()), tosca.TSTORE),
		),
		Effect: FailEffect(),
	})

	rules = append(rules, Rule{
		Name: "tstore_read_only",
		Condition: And(
			AnyKnownRevision(),
			Eq(Status(), st.Running),
			Eq(Op(Pc()), tosca.TSTORE),
			Eq(ReadOnly(), true),
		),
		Effect: FailEffect(),
	})

	// --- Stack PUSH0 ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.PUSH0,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R12_Shanghai, NewestSupportedRevision),
		},
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(0))
		},
	})...)

	rules = append(rules, []Rule{
		{
			Name: "push0_invalid_revision",
			Condition: And(
				RevisionBounds(tosca.R07_Istanbul, tosca.R11_Paris),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.PUSH0),
				Ge(Gas(), 2),
				Lt(StackSize(), st.MaxStackSize-1),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- MCOPY ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.MCOPY,
		staticGas: 3,
		pops:      3,
		pushes:    0,
		parameters: []Parameter{
			MemoryOffsetForCopyParameter{},
			MemoryOffsetForCopyParameter{},
			MemorySizeParameter{},
		},
		conditions: []Condition{
			RevisionBounds(tosca.R13_Cancun, NewestSupportedRevision),
		},
		effect: func(s *st.State) {
			destOffsetU256 := s.Stack.Pop()
			offsetU256 := s.Stack.Pop()
			sizeU256 := s.Stack.Pop()

			srcCost, srcOffset, size := s.Memory.ExpansionCosts(offsetU256, sizeU256)
			destCost, destOffset, _ := s.Memory.ExpansionCosts(destOffsetU256, sizeU256)
			wordCountCost := tosca.Gas(3 * ((size + 31) / 32))
			expansionCost := max(srcCost, destCost)

			dynamicGas, overflow := sumWithOverflow(expansionCost, wordCountCost)
			if s.Gas < dynamicGas || overflow {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= dynamicGas

			value := s.Memory.Read(srcOffset, size)
			s.Memory.Write(value, destOffset)
		},
	})...)

	rules = append(rules, []Rule{
		{
			Name: "mcopy_invalid_revision",
			Condition: And(
				RevisionBounds(tosca.R07_Istanbul, tosca.R12_Shanghai),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.MCOPY),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- Stack PUSH ---

	for i := 1; i <= 32; i++ {
		rules = append(rules, pushOp(i)...)
	}

	// --- Stack POP ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.POP,
		staticGas: 2,
		pops:      1,
		pushes:    0,
		effect: func(s *st.State) {
			s.Stack.Pop()
		},
	})...)

	// --- Stack DUP ---

	for i := 1; i <= 16; i++ {
		rules = append(rules, dupOp(i)...)
	}

	// --- Stack SWAP ---

	for i := 1; i <= 16; i++ {
		rules = append(rules, swapOp(i)...)
	}

	// --- LOG ---

	for i := 0; i <= 4; i++ {
		rules = append(rules, logOp(i)...)
	}

	// --- ADDRESS ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.ADDRESS,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256FromBytes(s.CallContext.AccountAddress[:]...))
		},
	})...)

	// --- ORIGIN ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.ORIGIN,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256FromBytes(s.TransactionContext.OriginAddress[:]...))
		},
	})...)

	// --- CALLER ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CALLER,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256FromBytes(s.CallContext.CallerAddress[:]...))
		},
	})...)

	// --- CALLVALUE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CALLVALUE,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(s.CallContext.Value)
		},
	})...)

	// --- NUMBER ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.NUMBER,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(s.BlockContext.BlockNumber))
		},
	})...)

	// --- BLOCKHASH ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.BLOCKHASH,
		staticGas: 20,
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			InRange256FromCurrentBlock(Param(0)),
		},
		parameters: []Parameter{NumericParameter{}},
		effect: func(s *st.State) {
			targetBlockNumber := s.Stack.Pop()
			index := s.BlockContext.BlockNumber - targetBlockNumber.Uint64()
			hash := s.RecentBlockHashes.Get(index - 1)
			s.Stack.Push(NewU256FromBytes(hash[:]...))
		},
	})...)

	rules = append(rules, rulesFor(instruction{
		op:        tosca.BLOCKHASH,
		name:      "_out_of_range",
		staticGas: 20,
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			OutOfRange256FromCurrentBlock(Param(0)),
		},
		parameters: []Parameter{NumericParameter{}},
		effect: func(s *st.State) {
			s.Stack.Pop()
			s.Stack.Push(NewU256(0))
		},
	})...)

	// --- COINBASE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.COINBASE,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256FromBytes(s.BlockContext.CoinBase[:]...))
		},
	})...)

	// --- GASLIMIT ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.GASLIMIT,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(s.BlockContext.GasLimit))
		},
	})...)

	// --- DIFFICULTY / PREVRANDAO ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.PREVRANDAO,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(s.BlockContext.PrevRandao)
		},
	})...)

	// --- GASPRICE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.GASPRICE,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(s.BlockContext.GasPrice)
		},
	})...)

	// --- EXTCODESIZE ---

	// cold
	rules = append(rules, rulesFor(instruction{
		op:        tosca.EXTCODESIZE,
		staticGas: 0 + 2600, // 2600 dynamic cost for cold address
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsAddressCold(Param(0)),
		},
		parameters: []Parameter{
			AddressParameter{},
		},
		effect: func(s *st.State) {
			address := NewAddress(s.Stack.Pop())
			size := s.Accounts.GetCode(address).Length()
			s.Stack.Push(NewU256(uint64(size)))
			s.Accounts.MarkWarm(address)
		},
		name: "_cold",
	})...)

	// warm
	rules = append(rules, rulesFor(instruction{
		op:        tosca.EXTCODESIZE,
		staticGas: 0 + 100, // 100 dynamic cost for warm address
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsAddressWarm(Param(0)),
		},
		parameters: []Parameter{
			AddressParameter{},
		},
		effect: func(s *st.State) {
			address := NewAddress(s.Stack.Pop())
			size := s.Accounts.GetCode(address).Length()
			s.Stack.Push(NewU256(uint64(size)))
		},
		name: "_warm",
	})...)

	// pre Berlin
	rules = append(rules, rulesFor(instruction{
		op:        tosca.EXTCODESIZE,
		staticGas: 700,
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			IsRevision(tosca.R07_Istanbul),
		},
		parameters: []Parameter{
			AddressParameter{},
		},
		effect: func(s *st.State) {
			address := NewAddress(s.Stack.Pop())
			size := s.Accounts.GetCode(address).Length()
			s.Stack.Push(NewU256(uint64(size)))
		},
		name: "_preBerlin",
	})...)

	// --- EXTCODECOPY ---

	// cold
	rules = append(rules, rulesFor(instruction{
		op:        tosca.EXTCODECOPY,
		staticGas: 2600,
		pops:      4,
		pushes:    0,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsAddressCold(Param(0)),
		},
		parameters: []Parameter{
			AddressParameter{},
			MemoryOffsetForCopyParameter{},
			DataOffsetParameter{},
			DataSizeParameter{}},
		effect: func(s *st.State) {
			extCodeCopyEffect(s, true)
		},
		name: "_cold",
	})...)

	// warm
	rules = append(rules, rulesFor(instruction{
		op:        tosca.EXTCODECOPY,
		staticGas: 100,
		pops:      4,
		pushes:    0,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsAddressWarm(Param(0)),
		},
		parameters: []Parameter{
			AddressParameter{},
			MemoryOffsetForCopyParameter{},
			DataOffsetParameter{},
			DataSizeParameter{}},
		effect: func(s *st.State) {
			extCodeCopyEffect(s, false)
		},
		name: "_warm",
	})...)

	// pre Berlin
	rules = append(rules, rulesFor(instruction{
		op:        tosca.EXTCODECOPY,
		staticGas: 700,
		pops:      4,
		pushes:    0,
		conditions: []Condition{
			IsRevision(tosca.R07_Istanbul),
		},
		parameters: []Parameter{
			AddressParameter{},
			MemoryOffsetForCopyParameter{},
			DataOffsetParameter{},
			DataSizeParameter{}},
		effect: func(s *st.State) {
			extCodeCopyEffect(s, false)
		},
		name: "_preBerlin",
	})...)

	// --- TIMESTAMP ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.TIMESTAMP,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(s.BlockContext.TimeStamp))
		},
	})...)

	// --- BASEFEE ---

	rules = append(rules, rulesFor(instruction{
		op:         tosca.BASEFEE,
		staticGas:  2,
		pops:       0,
		pushes:     1,
		conditions: []Condition{RevisionBounds(tosca.R10_London, NewestSupportedRevision)},
		effect: func(s *st.State) {
			s.Stack.Push(s.BlockContext.BaseFee)
		},
	})...)
	rules = append(rules, []Rule{
		{
			Name: "basefee_invalid_revision",
			Condition: And(
				RevisionBounds(tosca.R07_Istanbul, tosca.R09_Berlin),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.BASEFEE),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- BLOBHASH ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.BLOBHASH,
		staticGas: 3,
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R13_Cancun, NewestSupportedRevision),
			HasBlobHash(Param(0)),
		},
		parameters: []Parameter{NumericParameter{}},
		effect: func(s *st.State) {
			indexU256 := s.Stack.Pop()
			s.Stack.Push(NewU256FromBytes(s.TransactionContext.BlobHashes[indexU256.Uint64()][:]...))
		},
	})...)

	rules = append(rules, rulesFor(instruction{
		op:        tosca.BLOBHASH,
		name:      "_out_of_range",
		staticGas: 3,
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R13_Cancun, NewestSupportedRevision),
			HasNoBlobHash(Param(0)),
		},
		parameters: []Parameter{NumericParameter{}},
		effect: func(s *st.State) {
			s.Stack.Pop()
			s.Stack.Push(NewU256(0))
		},
	})...)

	rules = append(rules, []Rule{
		{
			Name: "blobhash_invalid_revision",
			Condition: And(
				RevisionBounds(tosca.R07_Istanbul, tosca.R12_Shanghai),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.BLOBHASH),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- BLOBBASEFEE ---

	rules = append(rules, rulesFor(instruction{
		op:         tosca.BLOBBASEFEE,
		staticGas:  2,
		pops:       0,
		pushes:     1,
		conditions: []Condition{RevisionBounds(tosca.R13_Cancun, NewestSupportedRevision)},
		effect: func(s *st.State) {
			s.Stack.Push(s.BlockContext.BlobBaseFee)
		},
	})...)

	rules = append(rules, []Rule{
		{
			Name: "blobbasefee_invalid_revision",
			Condition: And(
				RevisionBounds(tosca.R07_Istanbul, tosca.R12_Shanghai),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), tosca.BLOBBASEFEE),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- EXTCODEHASH ---

	extCodeHashEffect := func(s *st.State) tosca.Address {
		address := NewAddress(s.Stack.Pop())
		hash := s.Accounts.GetCodeHash(address)
		s.Stack.Push(NewU256FromBytes(hash[:]...))
		return address
	}

	// cold
	rules = append(rules, rulesFor(instruction{
		op:        tosca.EXTCODEHASH,
		name:      "_cold",
		staticGas: 0 + 2600, // 2600 dynamic cost for cold address
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsAddressCold(Param(0)),
		},
		parameters: []Parameter{
			AddressParameter{},
		},
		effect: func(s *st.State) {
			address := extCodeHashEffect(s)
			s.Accounts.MarkWarm(address)
		},
	})...)

	// warm
	rules = append(rules, rulesFor(instruction{
		op:        tosca.EXTCODEHASH,
		name:      "_warm",
		staticGas: 0 + 100, // 100 dynamic cost for warm address
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			RevisionBounds(tosca.R09_Berlin, NewestSupportedRevision),
			IsAddressWarm(Param(0)),
		},
		parameters: []Parameter{
			AddressParameter{},
		},
		effect: func(s *st.State) {
			extCodeHashEffect(s)
		},
	})...)

	// pre Berlin
	rules = append(rules, rulesFor(instruction{
		op:        tosca.EXTCODEHASH,
		name:      "_preBerlin",
		staticGas: 700,
		pops:      1,
		pushes:    1,
		conditions: []Condition{
			IsRevision(tosca.R07_Istanbul),
		},
		parameters: []Parameter{
			AddressParameter{},
		},
		effect: func(s *st.State) {
			extCodeHashEffect(s)
		},
	})...)

	// --- CHAINID ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CHAINID,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(s.BlockContext.ChainID)
		},
	})...)

	// --- CODESIZE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CODESIZE,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(uint64(s.Code.Length())))
		},
	})...)

	// --- CODECOPY ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CODECOPY,
		staticGas: 3,
		pops:      3,
		pushes:    0,
		parameters: []Parameter{
			MemoryOffsetForCopyParameter{},
			DataOffsetParameter{},
			DataSizeParameter{}},
		effect: func(s *st.State) {
			destOffsetU256 := s.Stack.Pop()
			offsetU256 := s.Stack.Pop()
			sizeU256 := s.Stack.Pop()

			cost, destOffset, size := s.Memory.ExpansionCosts(destOffsetU256, sizeU256)

			cost += tosca.Gas(3 * SizeInWords(size))
			if s.Gas < cost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= cost

			start := offsetU256.Uint64()
			if offsetU256.Gt(NewU256(uint64(s.Code.Length()))) {
				start = uint64(s.Code.Length())
			}
			end := min(start+size, uint64(s.Code.Length()))

			codeCopy := make([]byte, size)
			_ = s.Code.CopyCodeSlice(int(start), int(end), codeCopy)

			s.Memory.Write(codeCopy, destOffset)
		},
	})...)

	// --- CALLDATASIZE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CALLDATASIZE,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(uint64(s.CallData.Length())))
		},
	})...)

	// --- CALLDATALOAD ---

	rules = append(rules, rulesFor(instruction{
		op:         tosca.CALLDATALOAD,
		staticGas:  3,
		pops:       1,
		pushes:     1,
		parameters: []Parameter{MemoryOffsetForCopyParameter{}},
		effect: func(s *st.State) {
			offsetU256 := s.Stack.Pop()
			pushData := NewU256(0)

			len := s.CallData.Length()
			if offsetU256.IsUint64() {
				start := offsetU256.Uint64()
				if start > uint64(len) {
					start = uint64(len)
				}
				end := min(start+32, uint64(len))
				data := RightPadSlice(s.CallData.Get(start, end), 32)
				pushData = NewU256FromBytes(data...)
			}

			s.Stack.Push(pushData)
		},
	})...)

	// --- CALLDATACOPY ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CALLDATACOPY,
		staticGas: 3,
		pops:      3,
		pushes:    0,
		parameters: []Parameter{
			MemoryOffsetParameter{},
			DataOffsetParameter{},
			DataSizeParameter{}},
		effect: func(s *st.State) {
			destOffsetU256 := s.Stack.Pop()
			offsetU256 := s.Stack.Pop()
			sizeU256 := s.Stack.Pop()

			expansionCost, destOffset, size := s.Memory.ExpansionCosts(destOffsetU256, sizeU256)
			expansionCost += tosca.Gas(3 * SizeInWords(size))
			if s.Gas < expansionCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= expansionCost

			start := offsetU256.Uint64()
			len := s.CallData.Length()
			if offsetU256.Gt(NewU256(uint64(len))) {
				start = uint64(len)
			}
			end := min(start+size, uint64(len))
			dataBuffer := RightPadSlice(s.CallData.Get(start, end), int(size))
			s.Memory.Write(dataBuffer, destOffset)
		},
	})...)

	// --- SELFBALANCE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.SELFBALANCE,
		staticGas: 5,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			address := s.CallContext.AccountAddress
			balance := s.Accounts.GetBalance(address)
			s.Stack.Push(balance)
		},
	})...)

	// --- RETURNDATASIZE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.RETURNDATASIZE,
		staticGas: 2,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(uint64(s.LastCallReturnData.Length())))
		},
	})...)

	// --- RETURNDATACOPY ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.RETURNDATACOPY,
		staticGas: 3,
		pops:      3,
		pushes:    0,
		parameters: []Parameter{
			DataOffsetParameter{},
			DataOffsetParameter{},
			DataSizeParameter{}},
		effect: func(s *st.State) {
			destOffsetU256 := s.Stack.Pop()
			offsetU256 := s.Stack.Pop()
			sizeU256 := s.Stack.Pop()

			// offset + size overflows OR offset + size is larger than RETURNDATASIZE.
			offset := offsetU256.Uint64()
			readUntil := offset + sizeU256.Uint64()
			if !offsetU256.IsUint64() || !sizeU256.IsUint64() ||
				readUntil > uint64(s.LastCallReturnData.Length()) {
				s.Status = st.Failed
				s.Gas = 0
				return
			}

			expansionCost, destOffsetUint64, size := s.Memory.ExpansionCosts(destOffsetU256, sizeU256)
			expansionCost += tosca.Gas(3 * SizeInWords(size))
			if s.Gas < expansionCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= expansionCost

			s.Memory.Write(s.LastCallReturnData.Get(offset, readUntil), destOffsetUint64)
		},
	})...)

	// --- RETURN ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.RETURN,
		staticGas: 0,
		pops:      2,
		pushes:    0,
		parameters: []Parameter{
			MemoryOffsetParameter{},
			MemorySizeParameter{}},
		effect: func(s *st.State) {
			offsetU256 := s.Stack.Pop()
			sizeU256 := s.Stack.Pop()

			expansionCost, offset, size := s.Memory.ExpansionCosts(offsetU256, sizeU256)
			if s.Gas < expansionCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= expansionCost

			s.ReturnData = NewBytes(s.Memory.Read(offset, size))
			s.Status = st.Stopped
		},
	})...)

	// --- REVERT ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.REVERT,
		staticGas: 0,
		pops:      2,
		pushes:    0,
		parameters: []Parameter{
			MemoryOffsetParameter{},
			MemorySizeParameter{}},
		effect: func(s *st.State) {
			offsetU256 := s.Stack.Pop()
			sizeU256 := s.Stack.Pop()

			expansionCost, offset, size := s.Memory.ExpansionCosts(offsetU256, sizeU256)
			if s.Gas < expansionCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= expansionCost

			s.ReturnData = NewBytes(s.Memory.Read(offset, size))
			s.Status = st.Reverted
		},
	})...)

	// --- CALL, STATICCALL and DELEGATECALL ---

	rules = append(rules, getRulesForAllCallTypes()...)

	// --- SELFDESTRUCT ---

	for revision := tosca.R07_Istanbul; revision <= NewestSupportedRevision; revision++ {
		for _, warm := range []bool{true, false} {
			for _, hasSelfDestructed := range []bool{true, false} {
				coldTargetCost := tosca.Gas(0)
				createAccountFee := tosca.Gas(0)
				if !warm {
					createAccountFee = 25000
					if revision > tosca.R07_Istanbul {
						coldTargetCost = 2600
					}
				}
				rules = append(rules, nonStaticSelfDestructRules(revision, warm, coldTargetCost, createAccountFee, hasSelfDestructed)...)
			}
		}
	}

	rules = append(rules, rulesFor(instruction{
		op:        tosca.SELFDESTRUCT,
		name:      "_staticcall",
		staticGas: 5000,
		pops:      1,
		conditions: []Condition{
			Eq(ReadOnly(), true),
			AnyKnownRevision(),
		},
		effect: FailEffect().Apply,
	})...)

	// --- CREATE ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CREATE,
		name:      "_static",
		staticGas: 32000,
		pops:      3,
		pushes:    1,
		conditions: []Condition{
			Eq(ReadOnly(), true),
		},
		parameters: []Parameter{
			ValueParameter{},
			MemoryOffsetParameter{},
			MemorySizeParameter{},
		},
		effect: FailEffect().Apply,
	})...)

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CREATE,
		staticGas: 32000,
		pops:      3,
		pushes:    1,
		conditions: []Condition{
			Eq(ReadOnly(), false),
		},
		parameters: []Parameter{
			ValueParameter{},
			MemoryOffsetParameter{},
			MemorySizeParameter{},
		},
		effect: func(s *st.State) {
			createEffect(s, tosca.Create)
		},
	})...)

	// --- CREATE2 ---

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CREATE2,
		name:      "_static",
		staticGas: 32000,
		pops:      3,
		pushes:    1,
		conditions: []Condition{
			Eq(ReadOnly(), true),
		},
		parameters: []Parameter{
			ValueParameter{},
			MemoryOffsetParameter{},
			MemorySizeParameter{},
			NumericParameter{},
		},
		effect: FailEffect().Apply,
	})...)

	rules = append(rules, rulesFor(instruction{
		op:        tosca.CREATE2,
		staticGas: 32000,
		pops:      4,
		pushes:    1,
		conditions: []Condition{
			Eq(ReadOnly(), false),
		},
		parameters: []Parameter{
			ValueParameter{},
			MemoryOffsetParameter{},
			MemorySizeParameter{},
			NumericParameter{},
		},
		effect: func(s *st.State) {
			createEffect(s, tosca.Create2)
		},
	})...)

	// --- End ---

	return rules
}

func createEffect(s *st.State, callKind tosca.CallKind) {
	valueU256 := s.Stack.Pop()
	offsetU256 := s.Stack.Pop()
	sizeU256 := s.Stack.Pop()
	var saltU256 U256

	memExpCost, offset, size := s.Memory.ExpansionCosts(offsetU256, sizeU256)
	dynamicGas := memExpCost

	if s.Revision >= tosca.R12_Shanghai {
		const (
			MaxCodeSize     = 24576           // Maximum bytecode to permit for a contract
			MaxInitCodeSize = 2 * MaxCodeSize // Maximum initcode to permit in a creation transaction and create instructions

			InitCodeWordGas = 2 // Once per word of the init code when creating a contract.
		)
		if !sizeU256.IsUint64() || size > MaxInitCodeSize {
			s.Gas = 0
			s.Status = st.Failed
			return
		}
		dynamicGas += tosca.Gas(InitCodeWordGas * ((size + 31) / 32))
	}

	if callKind == tosca.Create2 {
		saltU256 = s.Stack.Pop()
		dynamicGas += tosca.Gas(6 * SizeInWords(size))
	}

	if s.Gas < dynamicGas {
		s.Status = st.Failed
		s.Gas = 0
		return
	}
	s.Gas -= dynamicGas
	input := s.Memory.Read(offset, size)

	if !valueU256.IsZero() {
		balance := s.Accounts.GetBalance(s.CallContext.AccountAddress)
		if balance.Lt(valueU256) {
			s.Stack.Push(AddressToU256(tosca.Address{}))
			s.LastCallReturnData = Bytes{}
			return
		}
	}

	limit := s.Gas - s.Gas/64

	res := s.CallJournal.Call(callKind, tosca.CallParameters{
		Sender: s.CallContext.AccountAddress,
		Value:  valueU256.Bytes32be(),
		Gas:    limit,
		Input:  input,
		Salt:   saltU256.Bytes32be(),
	})

	s.Gas -= limit - res.GasLeft
	s.GasRefund += res.GasRefund

	if !res.Success {
		s.Stack.Push(AddressToU256(tosca.Address{}))
		s.LastCallReturnData = NewBytes(res.Output)
		return
	}
	s.LastCallReturnData = Bytes{}
	s.Stack.Push(AddressToU256(res.CreatedAddress))
}

func binaryOpWithDynamicCost(
	op tosca.OpCode,
	costs tosca.Gas,
	effect func(a, b U256) U256,
	dynamicCost func(a, b U256) tosca.Gas,
) []Rule {
	return rulesFor(instruction{
		op:        op,
		staticGas: costs,
		pops:      2,
		pushes:    1,
		parameters: []Parameter{
			NumericParameter{},
			NumericParameter{},
		},
		effect: func(s *st.State) {
			a := s.Stack.Pop()
			b := s.Stack.Pop()
			dynamicCost := dynamicCost(a, b)
			// TODO: Improve handling of dynamic gas through dedicated constraint.
			if s.Gas < dynamicCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= dynamicCost
			s.Stack.Push(effect(a, b))
		},
	})
}

func binaryOp(
	op tosca.OpCode,
	costs tosca.Gas,
	effect func(a, b U256) U256,
) []Rule {
	return binaryOpWithDynamicCost(op, costs, effect, func(_, _ U256) tosca.Gas { return 0 })
}

func trinaryOp(
	op tosca.OpCode,
	costs tosca.Gas,
	effect func(a, b, c U256) U256,
) []Rule {
	return rulesFor(instruction{
		op:        op,
		staticGas: costs,
		pops:      3,
		pushes:    1,
		parameters: []Parameter{
			NumericParameter{},
			NumericParameter{},
			NumericParameter{},
		},
		effect: func(s *st.State) {
			a := s.Stack.Pop()
			b := s.Stack.Pop()
			c := s.Stack.Pop()
			s.Stack.Push(effect(a, b, c))
		},
	})
}

func unaryOp(
	op tosca.OpCode,
	costs tosca.Gas,
	effect func(a U256) U256,
) []Rule {
	return rulesFor(instruction{
		op:        op,
		staticGas: costs,
		pops:      1,
		pushes:    1,
		parameters: []Parameter{
			NumericParameter{},
		},
		effect: func(s *st.State) {
			a := s.Stack.Pop()
			s.Stack.Push(effect(a))
		},
	})
}

func pushOp(n int) []Rule {
	op := tosca.OpCode(int(tosca.PUSH1) + n - 1)
	return rulesFor(instruction{
		op:        op,
		staticGas: 3,
		pops:      0,
		pushes:    1,
		effect: func(s *st.State) {
			data := make([]byte, n)
			for i := 0; i < n; i++ {
				b, err := s.Code.GetData(int(s.Pc) + i)
				// This panic will never be triggered because the code generator always ensures that
				// after a PUSHX op there are X data bytes. This should be fixed by #592
				if err != nil {
					panic(err)
				}
				data[i] = b
			}
			s.Stack.Push(NewU256FromBytes(data...))
			s.Pc += uint16(n)
		},
	})
}

// An implementation does not necessarily do `n` pops and `n+1` pushes, since arbitrary stack positions could be accessed directly.
// However, the result is as if `n` pops and `n+1` pushes were performed.
func dupOp(n int) []Rule {
	op := tosca.OpCode(int(tosca.DUP1) + n - 1)
	return rulesFor(instruction{
		op:        op,
		staticGas: 3,
		pops:      n,
		pushes:    n + 1,
		effect: func(s *st.State) {
			s.Stack.Push(s.Stack.Get(n - 1))
		},
	})
}

// An implementation does not necessarily do `n` pops and `n+1` pushes, since arbitrary stack positions could be accessed directly.
// However, the result is as if `n` pops and `n+1` pushes were performed.
func swapOp(n int) []Rule {
	op := tosca.OpCode(int(tosca.SWAP1) + n - 1)
	return rulesFor(instruction{
		op:        op,
		staticGas: 3,
		pops:      n + 1,
		pushes:    n + 1,
		effect: func(s *st.State) {
			a := s.Stack.Get(0)
			b := s.Stack.Get(n)
			s.Stack.Set(0, b)
			s.Stack.Set(n, a)
		},
	})
}

func extCodeCopyEffect(s *st.State, markWarm bool) {
	address := NewAddress(s.Stack.Pop())
	destOffsetU256 := s.Stack.Pop()
	offsetU256 := s.Stack.Pop()
	sizeU256 := s.Stack.Pop()

	cost, destOffset, size := s.Memory.ExpansionCosts(destOffsetU256, sizeU256)
	cost += tosca.Gas(3 * SizeInWords(size))
	if s.Gas < cost {
		s.Status = st.Failed
		s.Gas = 0
		return
	}
	s.Gas -= cost

	start := offsetU256.Uint64()
	codeSize := uint64(s.Accounts.GetCode(address).Length())
	if offsetU256.Gt(NewU256(codeSize)) {
		start = codeSize
	}
	end := min(start+size, codeSize)

	codeCopy := RightPadSlice(s.Accounts.GetCode(address).ToBytes()[start:end], int(size))

	s.Memory.Write(codeCopy, destOffset)
	if markWarm {
		s.Accounts.MarkWarm(address)
	}
}

type sstoreOpParams struct {
	revision  tosca.Revision
	warm      bool
	config    tosca.StorageStatus
	gasCost   tosca.Gas
	gasRefund tosca.Gas
}

func sstoreOpRegular(params sstoreOpParams) Rule {
	name := fmt.Sprintf("sstore_regular_%v_%v", params.revision, params.config)

	gasLimit := tosca.Gas(2301) // EIP2200
	if params.gasCost > gasLimit {
		gasLimit = params.gasCost
	}

	conditions := []Condition{
		IsRevision(params.revision),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), tosca.SSTORE),
		Ge(Gas(), gasLimit),
		Eq(ReadOnly(), false),
		Ge(StackSize(), 2),
		StorageConfiguration(params.config, Param(0), Param(1)),
	}

	if params.revision >= tosca.R09_Berlin {
		if params.warm {
			name += "_warm"
			conditions = append(conditions, IsStorageWarm(Param(0)))
		} else {
			name += "_cold"
			conditions = append(conditions, IsStorageCold(Param(0)))
		}
	}

	return Rule{
		Name:      name,
		Condition: And(conditions...),
		Parameter: []Parameter{
			NumericParameter{},
			NumericParameter{},
		},
		Effect: Change(func(s *st.State) {
			s.GasRefund += params.gasRefund
			s.Gas -= params.gasCost
			s.Pc++
			key := s.Stack.Pop()
			value := s.Stack.Pop()
			s.Storage.SetCurrent(key, value)
			if s.Revision >= tosca.R09_Berlin {
				s.Storage.MarkWarm(key)
			}
		}),
	}
}

func sstoreOpTooLittleGas(params sstoreOpParams) Rule {
	name := fmt.Sprintf("sstore_with_too_little_gas_%v_%v", params.revision, params.config)

	conditions := []Condition{
		IsRevision(params.revision),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), tosca.SSTORE),
		Lt(Gas(), params.gasCost),
		Eq(ReadOnly(), false),
		Ge(StackSize(), 2),
		StorageConfiguration(params.config, Param(0), Param(1)),
	}

	if params.revision >= tosca.R09_Berlin {
		if params.warm {
			name += "_warm"
			conditions = append(conditions, IsStorageWarm(Param(0)))
		} else {
			name += "_cold"
			conditions = append(conditions, IsStorageCold(Param(0)))
		}
	}

	return Rule{
		Name:      name,
		Condition: And(conditions...),
		Parameter: []Parameter{
			NumericParameter{},
			NumericParameter{},
		},
		Effect: FailEffect(),
	}
}

func sstoreOpReadOnlyMode(params sstoreOpParams) Rule {
	name := fmt.Sprintf("sstore_in_read_only_mode_%v_%v", params.revision, params.config)

	gasLimit := tosca.Gas(2301) // EIP2200
	if params.gasCost > gasLimit {
		gasLimit = params.gasCost
	}

	conditions := []Condition{
		IsRevision(params.revision),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), tosca.SSTORE),
		Ge(Gas(), gasLimit),
		Eq(ReadOnly(), true),
		Ge(StackSize(), 2),
		StorageConfiguration(params.config, Param(0), Param(1)),
	}

	return Rule{
		Name:      name,
		Condition: And(conditions...),
		Parameter: []Parameter{
			NumericParameter{},
			NumericParameter{},
		},
		Effect: FailEffect(),
	}
}

func logOp(n int) []Rule {
	op := tosca.OpCode(int(tosca.LOG0) + n)
	minGas := tosca.Gas(375 + 375*n)
	conditions := []Condition{
		Eq(ReadOnly(), false),
	}

	parameter := []Parameter{
		MemoryOffsetParameter{},
		MemorySizeParameter{},
	}
	for i := 0; i < n; i++ {
		parameter = append(parameter, TopicParameter{})
	}

	rules := rulesFor(instruction{
		op:         op,
		staticGas:  minGas,
		pops:       2 + n,
		pushes:     0,
		conditions: conditions,
		parameters: parameter,
		effect: func(s *st.State) {
			offsetU256 := s.Stack.Pop()
			sizeU256 := s.Stack.Pop()

			topics := []U256{}
			for i := 0; i < n; i++ {
				topics = append(topics, s.Stack.Pop())
			}

			memExpCost, offset, size := s.Memory.ExpansionCosts(offsetU256, sizeU256)
			if s.Gas < memExpCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= memExpCost

			if s.Gas < tosca.Gas(8*size) {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= tosca.Gas(8 * size)

			s.Logs.AddLog(s.Memory.Read(offset, size), topics...)
		},
	})

	// Read only mode
	conditions = []Condition{
		AnyKnownRevision(),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), op),
		Ge(Gas(), minGas),
		Eq(ReadOnly(), true),
		Ge(StackSize(), 2+n),
	}

	rules = append(rules, []Rule{{
		Name:      fmt.Sprintf("%v_in_read_only_mode", strings.ToLower(op.String())),
		Condition: And(conditions...),
		Effect:    FailEffect(),
	}}...)

	return rules
}

func nonStaticSelfDestructRules(revision tosca.Revision, warm bool, destinationColdCost, accountCreationFee tosca.Gas, hasSelfDestructed bool) []Rule {

	var targetWarm Condition
	var warmColdString string
	if warm {
		warmColdString = "warm"
		targetWarm = IsAddressWarm(Param(0))
	} else {
		warmColdString = "cold"
		targetWarm = IsAddressCold(Param(0))
	}

	var hasSelfDestructedString string
	var hasSelfDestructedCondition Condition
	if hasSelfDestructed {
		hasSelfDestructedString = "destructed"
		hasSelfDestructedCondition = HasSelfDestructed()
	} else {
		hasSelfDestructedString = "not_destructed"
		hasSelfDestructedCondition = HasNotSelfDestructed()
	}

	refundGas := tosca.Gas(0)
	if revision < tosca.R10_London && !hasSelfDestructed {
		refundGas = 24000
	}

	name := fmt.Sprintf("_%v_%v_%v", strings.ToLower(revision.String()), warmColdString, hasSelfDestructedString)

	instruction := instruction{
		op:        tosca.SELFDESTRUCT,
		name:      name,
		staticGas: 5000,
		pops:      1,
		conditions: []Condition{
			Eq(ReadOnly(), false),
			IsRevision(revision),
			hasSelfDestructedCondition,
			targetWarm,
		},
		parameters: []Parameter{AddressParameter{}},
		effect: func(s *st.State) {
			selfDestructEffect(s, destinationColdCost, accountCreationFee, refundGas)
		},
	}

	return rulesFor(instruction)
}

func selfDestructEffect(s *st.State, destinationColdCost, accountCreationFee, refundGas tosca.Gas) {
	// Behavior pre cancun: the current account is registered to be destroyed, and will be at the end of the current
	// transaction. The transfer of the current balance to the given account cannot fail. In particular,
	// the destination account code (if any) is not executed, or, if the account does not exist, the
	// balance is still added to the given address.

	// account to send the current balance to
	destinationAccount := s.Stack.Pop().Bytes20be()
	currentAccount := s.CallContext.AccountAddress
	CurrentBalance := s.Accounts.GetBalance(currentAccount)

	dynamicCost := tosca.Gas(0)
	if !CurrentBalance.IsZero() && !s.Accounts.Exist(destinationAccount) {
		dynamicCost += accountCreationFee
	}

	dynamicCost += destinationColdCost

	if s.Gas < dynamicCost {
		s.Status = st.Failed
		return
	}
	s.Gas -= dynamicCost
	if s.Revision > tosca.R07_Istanbul {
		s.Accounts.MarkWarm(destinationAccount)
	}
	// add beneficiary to list in state
	s.HasSelfDestructed = true
	s.SelfDestructedJournal = append(s.SelfDestructedJournal, st.NewSelfDestructEntry(s.CallContext.AccountAddress, destinationAccount))
	s.Status = st.Stopped
	s.GasRefund += refundGas
}

func tooLittleGas(i instruction) []Rule {
	localConditions := append(i.conditions,
		AnyKnownRevision(),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), i.op),
		Lt(Gas(), i.staticGas))
	return []Rule{{
		Name:      fmt.Sprintf("%v_with_too_little_gas%v", strings.ToLower(i.op.String()), i.name),
		Condition: And(localConditions...),
		Effect:    FailEffect(),
	}}
}

func notEnoughSpace(i instruction) []Rule {
	localConditions := append(i.conditions,
		AnyKnownRevision(),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), i.op),
		Ge(StackSize(), st.MaxStackSize))
	return []Rule{{
		Name:      fmt.Sprintf("%v_with_not_enough_space%v", strings.ToLower(i.op.String()), i.name),
		Condition: And(localConditions...),
		Effect:    FailEffect(),
	}}
}

func tooFewElements(i instruction) []Rule {
	localConditions := append([]Condition{},
		AnyKnownRevision(),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), i.op),
		Lt(StackSize(), i.pops))
	return []Rule{{
		Name:      fmt.Sprintf("%v_with_too_few_elements%v", strings.ToLower(i.op.String()), i.name),
		Condition: And(localConditions...),
		Effect:    FailEffect(),
	}}
}

// rulesFor instantiates the basic rules depending on the instruction info.
// any rule that cannot be expressed using this function must be implemented manually.
// This function subtracts i.staticGas from state.Gas and increases state.Pc by one,
// these two are always done before calling i.effect. This should be kept
// in mind when implementing the effects of new rules.
func rulesFor(i instruction) []Rule {
	res := []Rule{}
	if i.staticGas > 0 {
		res = append(res, tooLittleGas(i)...)
	}
	if i.pops > 0 {
		res = append(res, tooFewElements(i)...)
	}
	if i.pushes > i.pops {
		res = append(res, notEnoughSpace(i)...)
	}
	localConditions := append(i.conditions,
		Eq(Status(), st.Running),
		Eq(Op(Pc()), i.op),
		Ge(Gas(), i.staticGas),
		Ge(StackSize(), i.pops),
		Le(StackSize(), st.MaxStackSize-(max(i.pushes-i.pops, 0))),
	)

	if !slices.ContainsFunc(i.conditions, IsRevisionCondition) {
		localConditions = append(localConditions, AnyKnownRevision())
	}

	res = append(res, Rule{
		Name:      fmt.Sprintf("%s_regular%v", strings.ToLower(i.op.String()), i.name),
		Condition: And(localConditions...),
		Parameter: i.parameters,
		Effect: Change(func(s *st.State) {
			s.Gas -= i.staticGas
			s.Pc++
			i.effect(s)
		}),
	})
	return res
}

// getRulesForAllCallTypes returns rules for CALL, CALLCODE, STATICCALL and DELEGATECALL
func getRulesForAllCallTypes() []Rule {
	// NOTE: this rule only covers Istanbul, Berlin and London cases in a coarse-grained way.
	// Follow-work is required to cover other revisions and situations,
	// as well as special cases currently covered in the effect function.
	callFailEffect := func(s *st.State, addrAccessCost tosca.Gas, op tosca.OpCode) {
		FailEffect().Apply(s)
	}

	res := []Rule{}
	for _, op := range []tosca.OpCode{tosca.CALL, tosca.CALLCODE, tosca.STATICCALL, tosca.DELEGATECALL} {
		for rev := tosca.R07_Istanbul; rev <= NewestSupportedRevision; rev++ {
			for _, warm := range []bool{true, false} {
				for _, static := range []bool{true, false} {
					for _, zeroValue := range []bool{true, false} {
						effect := callEffect
						if op == tosca.CALL && static && !zeroValue {
							effect = callFailEffect
						}
						res = append(res, getRulesForCall(op, rev, warm, zeroValue, effect, static)...)
					}
				}
			}
		}
	}

	return res
}

func getRulesForCall(op tosca.OpCode, revision tosca.Revision, warm, zeroValue bool, opEffect func(s *st.State, addrAccessCost tosca.Gas, op tosca.OpCode), static bool) []Rule {

	var staticGas tosca.Gas
	if revision == tosca.R07_Istanbul {
		staticGas = 700
	} else if revision == tosca.R09_Berlin {
		staticGas = 0
	}

	var addressAccessCost tosca.Gas
	if revision == tosca.R07_Istanbul {
		addressAccessCost = 0
	} else if revision >= tosca.R09_Berlin && warm {
		addressAccessCost = 100
	} else if revision >= tosca.R09_Berlin && !warm {
		addressAccessCost = 2600
	}

	var targetWarm Condition
	var warmColdString string
	if warm {
		warmColdString = "warm"
		targetWarm = IsAddressWarm(Param(1))
	} else {
		warmColdString = "cold"
		targetWarm = IsAddressCold(Param(1))
	}

	var staticCondition Condition
	var staticConditionName string
	if static {
		staticConditionName = "static"
		staticCondition = Eq(ReadOnly(), true)
	} else {
		staticConditionName = "not_static"
		staticCondition = Eq(ReadOnly(), false)
	}

	// default parameters, conditions and pops are for STATICCALL
	parameters := []Parameter{
		GasParameter{},
		AddressParameter{},
	}

	callConditions := []Condition{
		IsRevision(revision),
		targetWarm,
		staticCondition,
	}

	var valueZeroCondition Condition
	var valueZeroConditionName string
	var name string
	pops := 6

	if op == tosca.CALL || op == tosca.CALLCODE {
		parameters = append(parameters, ValueParameter{})

		if zeroValue {
			valueZeroConditionName = "_no_value"
			valueZeroCondition = Eq(ValueParam(2), NewU256(0))
		} else {
			valueZeroConditionName = "_with_value"
			valueZeroCondition = Ne(ValueParam(2), NewU256(0))
		}

		callConditions = append(callConditions, valueZeroCondition)

		pops = 7
	}

	parameters = append(parameters,
		MemoryOffsetParameter{},
		MemorySizeParameter{},
		MemoryOffsetParameter{},
		MemorySizeParameter{},
	)

	name = fmt.Sprintf("_%v_%v_%v%v", strings.ToLower(revision.String()), warmColdString,
		staticConditionName, valueZeroConditionName)

	return rulesFor(instruction{
		op:         op,
		name:       name,
		staticGas:  staticGas,
		pops:       pops,
		pushes:     1,
		conditions: callConditions,
		parameters: parameters,
		effect: func(s *st.State) {
			opEffect(s, addressAccessCost, op)
		},
	})
}

func callEffect(s *st.State, addrAccessCost tosca.Gas, op tosca.OpCode) {

	gas := s.Stack.Pop()
	target := s.Stack.Pop()
	var value U256
	if op == tosca.CALL || op == tosca.CALLCODE {
		value = s.Stack.Pop()
	}

	argsOffset := s.Stack.Pop()
	argsSize := s.Stack.Pop()
	retOffset := s.Stack.Pop()
	retSize := s.Stack.Pop()

	// --- dynamic costs ---

	// Compute the memory expansion costs of this call.
	inputMemoryExpansionCost, argsOffset64, argsSize64 := s.Memory.ExpansionCosts(argsOffset, argsSize)
	outputMemoryExpansionCost, retOffset64, retSize64 := s.Memory.ExpansionCosts(retOffset, retSize)
	memoryExpansionCost := inputMemoryExpansionCost
	if memoryExpansionCost < outputMemoryExpansionCost {
		memoryExpansionCost = outputMemoryExpansionCost
	}

	isValueZero := value.IsZero()

	// Compute the value transfer costs.
	positiveValueCost := tosca.Gas(0)
	if !isValueZero {
		positiveValueCost = 9000
	}

	// If an account is implicitly created, this costs extra.
	valueToEmptyAccountCost := tosca.Gas(0)
	if !isValueZero && s.Accounts.IsEmpty(target.Bytes20be()) && op != tosca.CALLCODE {
		valueToEmptyAccountCost = 25000
	}

	// Deduct the gas costs for this call, except the costs for the recursive call.
	dynamicGas, overflow := sumWithOverflow(memoryExpansionCost, positiveValueCost, valueToEmptyAccountCost, addrAccessCost)
	if s.Gas < dynamicGas || overflow {
		s.Status = st.Failed
		return
	}
	s.Gas -= dynamicGas

	if s.Revision >= tosca.R09_Berlin {
		s.Accounts.MarkWarm(target.Bytes20be())
	}

	// Grow the memory for which gas has just been deducted.
	s.Memory.Grow(argsOffset64, argsSize64)
	s.Memory.Grow(retOffset64, retSize64)

	// Compute the gas provided to the nested call.
	limit := s.Gas - s.Gas/64
	endowment := limit
	if gas.IsUint64() && gas.Uint64() < uint64(endowment) {
		endowment = tosca.Gas(gas.Uint64())
	}

	// If value is transferred, a stipend is granted.
	stipend := tosca.Gas(0)
	if !isValueZero {
		stipend = 2300
	}
	s.Gas += stipend

	// Read the input from the call from memory.
	input := s.Memory.Read(argsOffset64, argsSize64)

	// --- call execution ---

	// Check that the caller has enough balance to transfer the requested value.
	if !isValueZero {
		balance := s.Accounts.GetBalance(s.CallContext.AccountAddress)
		if balance.Lt(value) {
			s.Stack.Push(NewU256(0))
			s.LastCallReturnData = Bytes{}
			return
		}
	}

	sender := s.CallContext.AccountAddress
	recipient := target.Bytes20be()
	codeAddress := tosca.Address{}
	// In a static context all calls are static calls.
	kind := tosca.Call
	if op == tosca.DELEGATECALL {
		kind = tosca.DelegateCall
		sender = s.CallContext.CallerAddress
		recipient = s.CallContext.AccountAddress
		codeAddress = target.Bytes20be()
		value = s.CallContext.Value
	} else if op == tosca.CALLCODE {
		kind = tosca.CallCode
		sender = s.CallContext.AccountAddress
		recipient = s.CallContext.AccountAddress
		codeAddress = target.Bytes20be()
	}

	if (s.ReadOnly && op == tosca.CALL) || op == tosca.STATICCALL {
		kind = tosca.StaticCall
	}

	// Execute the call.
	res := s.CallJournal.Call(kind, tosca.CallParameters{
		Sender:      sender,
		Recipient:   recipient,
		Value:       value.Bytes32be(),
		Gas:         endowment + stipend,
		Input:       input,
		CodeAddress: codeAddress,
	})

	// Process the result.
	if retSize64 > 0 {
		output := res.Output
		if len(output) > int(retSize64) {
			output = output[0:retSize64]
		}
		s.Memory.Write(output, retOffset64)
	}

	s.Gas -= endowment + stipend - res.GasLeft // < the costs for the code execution
	s.GasRefund += res.GasRefund
	s.LastCallReturnData = NewBytes(res.Output)
	if res.Success {
		s.Stack.Push(NewU256(1))
	} else {
		s.Stack.Push(NewU256(0))
	}
}

func sumWithOverflow(values ...tosca.Gas) (tosca.Gas, bool) {
	res := tosca.Gas(0)
	for _, cur := range values {
		next := res + cur
		if next < res {
			return 0, true
		}
		res = next
	}
	return res, false
}
