package spc

import (
	"fmt"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/gen"
	. "github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// Specification defines the interface for handling specifications.
type Specification interface {
	// GetRules provides access to all rules within the specification.
	GetRules() []Rule

	// GetRulesFor gives you all rules that apply to the given State (i.e. where
	// the rule's Condition holds).
	GetRulesFor(*st.State) []Rule
}

type specification struct {
	rules []Rule
}

func NewSpecification(rules ...Rule) Specification {
	return &specification{rules}
}

func (s *specification) GetRules() []Rule {
	return s.rules
}

func (s *specification) GetRulesFor(state *st.State) []Rule {
	result := []Rule{}
	for _, rule := range s.rules {
		if valid, err := rule.Condition.Check(state); valid && err == nil {
			result = append(result, rule)
		}
	}
	return result
}

// instruction holds the basic information for the 4 basic rules
// these are not enough gas, stack overflow, stack underflow, and a regular behaviour case
type instruction struct {
	op         OpCode
	static_gas uint64
	pops       int
	pushes     int
	conditions []Condition       // conditions for the regular case
	parameters []Parameter       // parameters for the regular case
	effect     func(s *st.State) // effect for the regular case
	name       string
}

func noEffect(st *st.State) {}

////////////////////////////////////////////////////////////

func boolToU256(value bool) U256 {
	if value {
		return NewU256(1)
	}
	return NewU256(0)
}

var Spec = func() Specification {
	rules := []Rule{}

	// --- Terminal States ---

	rules = append(rules, []Rule{
		{
			Name:      "stopped_is_end",
			Condition: And(AnyKnownRevision(), Eq(Status(), st.Stopped)),
			Effect:    NoEffect(),
		},

		{
			Name:      "returned_is_end",
			Condition: And(AnyKnownRevision(), Eq(Status(), st.Returned)),
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
	}...)

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
			Eq(Op(Pc()), STOP),
		),
		Effect: Change(func(s *st.State) {
			s.Status = st.Stopped
			s.Pc++
		}),
	})

	// --- Arithmetic ---

	rules = append(rules, binaryOp(ADD, 3, func(a, b U256) U256 {
		return a.Add(b)
	})...)

	rules = append(rules, binaryOp(MUL, 5, func(a, b U256) U256 {
		return a.Mul(b)
	})...)

	rules = append(rules, binaryOp(SUB, 3, func(a, b U256) U256 {
		return a.Sub(b)
	})...)

	rules = append(rules, binaryOp(DIV, 5, func(a, b U256) U256 {
		return a.Div(b)
	})...)

	rules = append(rules, binaryOp(SDIV, 5, func(a, b U256) U256 {
		return a.SDiv(b)
	})...)

	rules = append(rules, binaryOp(MOD, 5, func(a, b U256) U256 {
		return a.Mod(b)
	})...)

	rules = append(rules, binaryOp(SMOD, 5, func(a, b U256) U256 {
		return a.SMod(b)
	})...)

	rules = append(rules, trinaryOp(ADDMOD, 8, func(a, b, n U256) U256 {
		return a.AddMod(b, n)
	})...)

	rules = append(rules, trinaryOp(MULMOD, 8, func(a, b, n U256) U256 {
		return a.MulMod(b, n)
	})...)

	rules = append(rules, binaryOpWithDynamicCost(EXP, 10, func(a, e U256) U256 {
		return a.Exp(e)
	}, func(a, e U256) uint64 {
		const gasFactor = uint64(50)
		expBytes := e.Bytes32be()
		for i := 0; i < 32; i++ {
			if expBytes[i] != 0 {
				return gasFactor * uint64(32-i)
			}
		}
		return 0
	})...)

	rules = append(rules, binaryOp(SIGNEXTEND, 5, func(b, x U256) U256 {
		return x.SignExtend(b)
	})...)

	rules = append(rules, binaryOp(LT, 3, func(a, b U256) U256 {
		return boolToU256(a.Lt(b))
	})...)

	rules = append(rules, binaryOp(GT, 3, func(a, b U256) U256 {
		return boolToU256(a.Gt(b))
	})...)

	rules = append(rules, binaryOp(SLT, 3, func(a, b U256) U256 {
		return boolToU256(a.Slt(b))
	})...)

	rules = append(rules, binaryOp(SGT, 3, func(a, b U256) U256 {
		return boolToU256(a.Sgt(b))
	})...)

	rules = append(rules, binaryOp(EQ, 3, func(a, b U256) U256 {
		return boolToU256(a.Eq(b))
	})...)

	rules = append(rules, unaryOp(ISZERO, 3, func(a U256) U256 {
		return boolToU256(a.IsZero())
	})...)

	rules = append(rules, binaryOp(AND, 3, func(a, b U256) U256 {
		return a.And(b)
	})...)

	rules = append(rules, binaryOp(OR, 3, func(a, b U256) U256 {
		return a.Or(b)
	})...)

	rules = append(rules, binaryOp(XOR, 3, func(a, b U256) U256 {
		return a.Xor(b)
	})...)

	rules = append(rules, unaryOp(NOT, 3, func(a U256) U256 {
		return a.Not()
	})...)

	rules = append(rules, binaryOp(BYTE, 3, func(i, x U256) U256 {
		if i.Gt(NewU256(31)) {
			return NewU256(0)
		}
		return NewU256(uint64(x.Bytes32be()[i.Uint64()]))
	})...)

	rules = append(rules, binaryOp(SHL, 3, func(shift, value U256) U256 {
		return value.Shl(shift)
	})...)

	rules = append(rules, binaryOp(SHR, 3, func(shift, value U256) U256 {
		return value.Shr(shift)
	})...)

	rules = append(rules, binaryOp(SAR, 3, func(shift, value U256) U256 {
		return value.Srsh(shift)
	})...)

	// --- SHA3 ---

	rules = append(rules, rulesFor(instruction{
		op:         SHA3,
		static_gas: 30,
		pops:       2,
		pushes:     1,
		parameters: []Parameter{
			MemoryOffsetParameter{},
			MemorySizeParameter{},
		},
		effect: func(s *st.State) {
			offset_u256 := s.Stack.Pop()
			size_u256 := s.Stack.Pop()

			memExpCost, offset, size := s.Memory.ExpansionCosts(offset_u256, size_u256)
			if s.Gas < memExpCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= memExpCost

			wordCost := 6 * ((size + 31) / 32)
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

	// --- MLOAD ---

	rules = append(rules, rulesFor(instruction{
		op:         MLOAD,
		static_gas: 3,
		pops:       1,
		pushes:     1,
		parameters: []Parameter{
			MemoryOffsetParameter{},
		},
		effect: func(s *st.State) {
			offset_u256 := s.Stack.Pop()

			cost, offset, _ := s.Memory.ExpansionCosts(offset_u256, NewU256(32))
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
		op:         MSTORE,
		static_gas: 3,
		pops:       2,
		pushes:     0,
		parameters: []Parameter{
			MemoryOffsetParameter{},
			NumericParameter{},
		},
		effect: func(s *st.State) {
			offset_u256 := s.Stack.Pop()
			value := s.Stack.Pop()

			cost, offset, _ := s.Memory.ExpansionCosts(offset_u256, NewU256(32))
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
		op:         MSTORE8,
		static_gas: 3,
		pops:       2,
		pushes:     0,
		parameters: []Parameter{
			MemoryOffsetParameter{},
			NumericParameter{},
		},
		effect: func(s *st.State) {
			offset_u256 := s.Stack.Pop()
			value := s.Stack.Pop()

			cost, offset, _ := s.Memory.ExpansionCosts(offset_u256, NewU256(1))
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
		op:         SLOAD,
		static_gas: 100 + 2000, // 2000 are from the dynamic cost of cold mem
		pops:       1,
		pushes:     1,
		conditions: []Condition{
			RevisionBounds(R09_Berlin, R10_London),
			IsStorageCold(Param(0)),
		},
		parameters: []Parameter{
			NumericParameter{},
		},
		effect: func(s *st.State) {
			key := s.Stack.Pop()
			s.Stack.Push(s.Storage.Current[key])
			s.Storage.MarkWarm(key)
		},
		name: "_cold",
	})...)

	// warm
	rules = append(rules, rulesFor(instruction{
		op:         SLOAD,
		static_gas: 100,
		pops:       1,
		pushes:     1,
		conditions: []Condition{
			RevisionBounds(R09_Berlin, R10_London),
			IsStorageWarm(Param(0)),
		},
		parameters: []Parameter{
			NumericParameter{},
		},
		effect: func(s *st.State) {
			key := s.Stack.Pop()
			s.Stack.Push(s.Storage.Current[key])
		},
		name: "_warm",
	})...)

	// pre_berlin
	rules = append(rules, rulesFor(instruction{
		op:         SLOAD,
		static_gas: 800,
		pops:       1,
		pushes:     1,
		conditions: []Condition{
			RevisionBounds(R07_Istanbul, R07_Istanbul),
		},
		parameters: []Parameter{
			NumericParameter{},
		},
		effect: func(s *st.State) {
			key := s.Stack.Pop()
			s.Stack.Push(s.Storage.Current[key])
		},
		name: "_pre_berlin",
	})...)

	// --- SSTORE ---

	sstoreRules := []sstoreOpParams{
		{revision: R07_Istanbul, config: gen.StorageAssigned, gasCost: 800},
		{revision: R07_Istanbul, config: gen.StorageAdded, gasCost: 20000},
		{revision: R07_Istanbul, config: gen.StorageAddedDeleted, gasCost: 800, gasRefund: 19200},
		{revision: R07_Istanbul, config: gen.StorageDeletedRestored, gasCost: 800, gasRefund: -10800},
		{revision: R07_Istanbul, config: gen.StorageDeletedAdded, gasCost: 800, gasRefund: -15000},
		{revision: R07_Istanbul, config: gen.StorageDeleted, gasCost: 5000, gasRefund: 15000},
		{revision: R07_Istanbul, config: gen.StorageModified, gasCost: 5000},
		{revision: R07_Istanbul, config: gen.StorageModifiedDeleted, gasCost: 800, gasRefund: 15000},
		{revision: R07_Istanbul, config: gen.StorageModifiedRestored, gasCost: 800, gasRefund: 4200},

		// Certain storage configurations imply warm access. Not all
		// combinations are possible; invalid ones are marked below.

		// {revision: R09_Berlin, warm: false, config: gen.StorageAssigned, gasCost: 2200}, // invalid
		{revision: R09_Berlin, warm: false, config: gen.StorageAdded, gasCost: 22100},
		// {revision: R09_Berlin, warm: false, config: gen.StorageAddedDeleted, gasCost: 2200, gasRefund: 19900},     // invalid
		// {revision: R09_Berlin, warm: false, config: gen.StorageDeletedRestored, gasCost: 2200, gasRefund: -10800}, // invalid
		// {revision: R09_Berlin, warm: false, config: gen.StorageDeletedAdded, gasCost: 2200, gasRefund: -15000},    // invalid
		{revision: R09_Berlin, warm: false, config: gen.StorageDeleted, gasCost: 5000, gasRefund: 15000},
		{revision: R09_Berlin, warm: false, config: gen.StorageModified, gasCost: 5000},
		// {revision: R09_Berlin, warm: false, config: gen.StorageModifiedDeleted, gasCost: 2200, gasRefund: 15000}, // invalid
		// {revision: R09_Berlin, warm: false, config: gen.StorageModifiedRestored, gasCost: 2200, gasRefund: 4900}, // invalid

		{revision: R09_Berlin, warm: true, config: gen.StorageAssigned, gasCost: 100},
		{revision: R09_Berlin, warm: true, config: gen.StorageAdded, gasCost: 20000},
		{revision: R09_Berlin, warm: true, config: gen.StorageAddedDeleted, gasCost: 100, gasRefund: 19900},
		{revision: R09_Berlin, warm: true, config: gen.StorageDeletedRestored, gasCost: 100, gasRefund: -12200},
		{revision: R09_Berlin, warm: true, config: gen.StorageDeletedAdded, gasCost: 100, gasRefund: -15000},
		{revision: R09_Berlin, warm: true, config: gen.StorageDeleted, gasCost: 2900, gasRefund: 15000},
		{revision: R09_Berlin, warm: true, config: gen.StorageModified, gasCost: 2900},
		{revision: R09_Berlin, warm: true, config: gen.StorageModifiedDeleted, gasCost: 100, gasRefund: 15000},
		{revision: R09_Berlin, warm: true, config: gen.StorageModifiedRestored, gasCost: 100, gasRefund: 2800},

		// {revision: R10_London, warm: false, config: gen.StorageAssigned, gasCost: 2200}, // invalid
		{revision: R10_London, warm: false, config: gen.StorageAdded, gasCost: 22100},
		// {revision: R10_London, warm: false, config: gen.StorageAddedDeleted, gasCost: 2200, gasRefund: 19900},  // invalid
		// {revision: R10_London, warm: false, config: gen.StorageDeletedRestored, gasCost: 2200, gasRefund: 100}, // invalid
		// {revision: R10_London, warm: false, config: gen.StorageDeletedAdded, gasCost: 2200, gasRefund: -4800},  // invalid
		{revision: R10_London, warm: false, config: gen.StorageDeleted, gasCost: 5000, gasRefund: 4800},
		{revision: R10_London, warm: false, config: gen.StorageModified, gasCost: 5000},
		// {revision: R10_London, warm: false, config: gen.StorageModifiedDeleted, gasCost: 2200, gasRefund: 4800},  // invalid
		// {revision: R10_London, warm: false, config: gen.StorageModifiedRestored, gasCost: 2200, gasRefund: 4900}, // invalid

		{revision: R10_London, warm: true, config: gen.StorageAssigned, gasCost: 100},
		{revision: R10_London, warm: true, config: gen.StorageAdded, gasCost: 20000},
		{revision: R10_London, warm: true, config: gen.StorageAddedDeleted, gasCost: 100, gasRefund: 19900},
		{revision: R10_London, warm: true, config: gen.StorageDeletedRestored, gasCost: 100, gasRefund: -2000},
		{revision: R10_London, warm: true, config: gen.StorageDeletedAdded, gasCost: 100, gasRefund: -4800},
		{revision: R10_London, warm: true, config: gen.StorageDeleted, gasCost: 2900, gasRefund: 4800},
		{revision: R10_London, warm: true, config: gen.StorageModified, gasCost: 2900},
		{revision: R10_London, warm: true, config: gen.StorageModifiedDeleted, gasCost: 100, gasRefund: 4800},
		{revision: R10_London, warm: true, config: gen.StorageModifiedRestored, gasCost: 100, gasRefund: 2800},
	}
	for _, params := range sstoreRules {
		rules = append(rules, sstoreOpRegular(params))
		rules = append(rules, sstoreOpTooLittleGas(params))
	}

	rules = append(rules, tooLittleGas(instruction{op: SSTORE, static_gas: 2300, name: "_EIP2200"})...)
	rules = append(rules, tooFewElements(instruction{op: SSTORE, static_gas: 2, pops: 2})...)

	// --- JUMP ---

	rules = append(rules, rulesFor(instruction{
		op:         JUMP,
		static_gas: 8,
		pops:       1,
		pushes:     0,
		conditions: []Condition{
			IsCode(Param(0)),
			Eq(Op(Param(0)), JUMPDEST),
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
				Eq(Op(Pc()), JUMP),
				Ge(Gas(), 8),
				Ge(StackSize(), 1),
				IsData(Param(0)),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jump_to_invalid_destination",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Ge(Gas(), 8),
				Ge(StackSize(), 1),
				IsCode(Param(0)),
				Ne(Op(Param(0)), JUMPDEST),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- JUMPI ---

	rules = append(rules, rulesFor(instruction{
		op:         JUMPI,
		static_gas: 10,
		pops:       2,
		pushes:     0,
		conditions: []Condition{
			IsCode(Param(0)),
			Eq(Op(Param(0)), JUMPDEST),
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
				Eq(Op(Pc()), JUMPI),
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
				Eq(Op(Pc()), JUMPI),
				Ge(Gas(), 10),
				Ge(StackSize(), 2),
				IsData(Param(0)),
				Ne(Param(1), NewU256(0)),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jumpi_to_invalid_destination",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Ge(Gas(), 10),
				Ge(StackSize(), 2),
				IsCode(Param(0)),
				Ne(Op(Param(0)), JUMPDEST),
				Ne(Param(1), NewU256(0)),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- PC ---

	rules = append(rules, rulesFor(instruction{
		op:         PC,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(uint64(s.Pc) - 1))
		},
	})...)

	// --- JUMPDEST ---

	rules = append(rules, rulesFor(instruction{
		op:         JUMPDEST,
		static_gas: 1,
		pops:       0,
		pushes:     0,
		effect:     noEffect,
	})...)

	// --- Stack PUSH ---

	for i := 1; i <= 32; i++ {
		rules = append(rules, pushOp(i)...)
	}

	// --- Stack POP ---

	rules = append(rules, rulesFor(instruction{
		op:         POP,
		static_gas: 2,
		pops:       1,
		pushes:     0,
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
		op:         ADDRESS,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256FromBytes(s.CallContext.AccountAddress[:]...))
		},
	})...)

	// --- ORIGIN ---

	rules = append(rules, rulesFor(instruction{
		op:         ORIGIN,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256FromBytes(s.CallContext.OriginAddress[:]...))
		},
	})...)

	// --- CALLER ---

	rules = append(rules, rulesFor(instruction{
		op:         CALLER,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256FromBytes(s.CallContext.CallerAddress[:]...))
		},
	})...)

	// --- CALLVALUE ---

	rules = append(rules, rulesFor(instruction{
		op:         CALLVALUE,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(s.CallContext.Value)
		},
	})...)

	// --- NUMBER ---

	rules = append(rules, rulesFor(instruction{
		op:         NUMBER,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(s.BlockContext.BlockNumber))
		},
	})...)

	// --- COINBASE ---

	rules = append(rules, rulesFor(instruction{
		op:         COINBASE,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256FromBytes(s.BlockContext.CoinBase[:]...))
		},
	})...)

	// --- GASLIMIT ---

	rules = append(rules, rulesFor(instruction{
		op:         GASLIMIT,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(s.BlockContext.GasLimit))
		},
	})...)

	// --- DIFFICULTY ---

	rules = append(rules, rulesFor(instruction{
		op:         DIFFICULTY,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(s.BlockContext.Difficulty)
		},
	})...)

	// --- GASPRICE ---

	rules = append(rules, rulesFor(instruction{
		op:         GASPRICE,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(s.BlockContext.GasPrice)
		},
	})...)

	// --- TIMESTAMP ---

	rules = append(rules, rulesFor(instruction{
		op:         TIMESTAMP,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(NewU256(s.BlockContext.TimeStamp))
		},
	})...)

	// --- BASEFEE ---

	rules = append(rules, rulesFor(instruction{
		op:         BASEFEE,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		conditions: []Condition{IsRevision(R10_London)},
		effect: func(s *st.State) {
			s.Stack.Push(s.BlockContext.BaseFee)
		},
	})...)
	rules = append(rules, []Rule{
		{
			Name: "basefee_invalid_revision",
			Condition: And(
				RevisionBounds(R07_Istanbul, R09_Berlin),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), BASEFEE),
				Ge(Gas(), 2),
				Lt(StackSize(), st.MaxStackSize),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- CHAINID ---

	rules = append(rules, rulesFor(instruction{
		op:         CHAINID,
		static_gas: 2,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			s.Stack.Push(s.BlockContext.ChainID)
		},
	})...)

	// --- End ---

	return NewSpecification(rules...)
}()

func binaryOpWithDynamicCost(
	op OpCode,
	costs uint64,
	effect func(a, b U256) U256,
	dynamicCost func(a, b U256) uint64,
) []Rule {
	return rulesFor(instruction{
		op:         op,
		static_gas: costs,
		pops:       2,
		pushes:     1,
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
	op OpCode,
	costs uint64,
	effect func(a, b U256) U256,
) []Rule {
	return binaryOpWithDynamicCost(op, costs, effect, func(_, _ U256) uint64 { return 0 })
}

func trinaryOp(
	op OpCode,
	costs uint64,
	effect func(a, b, c U256) U256,
) []Rule {
	return rulesFor(instruction{
		op:         op,
		static_gas: costs,
		pops:       3,
		pushes:     1,
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
	op OpCode,
	costs uint64,
	effect func(a U256) U256,
) []Rule {
	return rulesFor(instruction{
		op:         op,
		static_gas: costs,
		pops:       1,
		pushes:     1,
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
	op := OpCode(int(PUSH1) + n - 1)
	return rulesFor(instruction{
		op:         op,
		static_gas: 3,
		pops:       0,
		pushes:     1,
		effect: func(s *st.State) {
			data := make([]byte, n)
			for i := 0; i < n; i++ {
				b, err := s.Code.GetData(int(s.Pc) + i)
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
	op := OpCode(int(DUP1) + n - 1)
	return rulesFor(instruction{
		op:         op,
		static_gas: 3,
		pops:       n,
		pushes:     n + 1,
		effect: func(s *st.State) {
			s.Stack.Push(s.Stack.Get(n - 1))
		},
	})
}

// An implementation does not necessarily do `n` pops and `n+1` pushes, since arbitrary stack positions could be accessed directly.
// However, the result is as if `n` pops and `n+1` pushes were performed.
func swapOp(n int) []Rule {
	op := OpCode(int(SWAP1) + n - 1)
	return rulesFor(instruction{
		op:         op,
		static_gas: 3,
		pops:       n + 1,
		pushes:     n + 1,
		effect: func(s *st.State) {
			a := s.Stack.Get(0)
			b := s.Stack.Get(n)
			s.Stack.Set(0, b)
			s.Stack.Set(n, a)
		},
	})
}

type sstoreOpParams struct {
	revision  Revision
	warm      bool
	config    gen.StorageCfg
	gasCost   uint64
	gasRefund int64
}

func sstoreOpRegular(params sstoreOpParams) Rule {
	name := fmt.Sprintf("sstore_regular_%v_%v", params.revision, params.config)

	gasLimit := uint64(2301) // EIP2200
	if params.gasCost > gasLimit {
		gasLimit = params.gasCost
	}

	conditions := []Condition{
		IsRevision(params.revision),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), SSTORE),
		Ge(Gas(), gasLimit),
		Ge(StackSize(), 2),
		StorageConfiguration(params.config, Param(0), Param(1)),
	}

	if params.revision >= R09_Berlin {
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
			if params.gasRefund < 0 {
				if s.GasRefund < uint64(-params.gasRefund) {
					// Gas refund must not become negative!
					s.Status = st.Failed
					s.Gas = 0
					return
				}
				s.GasRefund -= uint64(-params.gasRefund)
			} else {
				s.GasRefund += uint64(params.gasRefund)
			}

			s.Gas -= params.gasCost
			s.Pc++
			key := s.Stack.Pop()
			value := s.Stack.Pop()
			s.Storage.Current[key] = value
			if s.Revision >= R09_Berlin {
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
		Eq(Op(Pc()), SSTORE),
		Lt(Gas(), params.gasCost),
		Ge(StackSize(), 2),
		StorageConfiguration(params.config, Param(0), Param(1)),
	}

	if params.revision >= R09_Berlin {
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

func logOp(n int) []Rule {
	op := OpCode(int(LOG0) + n)
	minGas := uint64(375 + 375*n)

	parameter := []Parameter{
		MemoryOffsetParameter{},
		MemorySizeParameter{},
	}
	for i := 0; i < n; i++ {
		parameter = append(parameter, TopicParameter{})
	}

	return rulesFor(instruction{
		op:         op,
		static_gas: minGas,
		pops:       2 + n,
		pushes:     0,
		parameters: parameter,
		effect: func(s *st.State) {
			offset_u256 := s.Stack.Pop()
			size_u256 := s.Stack.Pop()

			topics := []U256{}
			for i := 0; i < n; i++ {
				topics = append(topics, s.Stack.Pop())
			}

			memExpCost, offset, size := s.Memory.ExpansionCosts(offset_u256, size_u256)
			if s.Gas < memExpCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= memExpCost

			if s.Gas < 8*size {
				s.Status = st.Failed
				s.Gas = 0
				return
			}
			s.Gas -= 8 * size

			s.Logs.AddLog(s.Memory.Read(offset, size), topics...)
		},
	})
}

func tooLittleGas(i instruction) []Rule {
	localConditions := append(i.conditions,
		AnyKnownRevision(),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), i.op),
		Lt(Gas(), i.static_gas))
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
// This function subtracts i.static_gas from state.Gas and increases state.Pc by one,
// these two are always done before calling i.effect. This should be kept
// in mind when implementing the effects of new rules.
func rulesFor(i instruction) []Rule {
	res := []Rule{}
	res = append(res, tooLittleGas(i)...)
	if i.pops > 0 {
		res = append(res, tooFewElements(i)...)
	}
	if i.pushes > i.pops {
		res = append(res, notEnoughSpace(i)...)
	}
	localConditions := append(i.conditions,
		AnyKnownRevision(),
		Eq(Status(), st.Running),
		Eq(Op(Pc()), i.op),
		Ge(Gas(), i.static_gas),
		Ge(StackSize(), i.pops),
		Le(StackSize(), st.MaxStackSize-(max(i.pushes-i.pops, 0))),
	)

	res = append(res, []Rule{
		{
			Name:      fmt.Sprintf("%s_regular%v", strings.ToLower(i.op.String()), i.name),
			Condition: And(localConditions...),
			Parameter: i.parameters,
			Effect: Change(func(s *st.State) {
				s.Gas -= i.static_gas
				s.Pc++
				i.effect(s)
			}),
		},
	}...)
	return res
}
