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
	rules = append(rules, []Rule{
		{
			Name: "sha3_with_too_little_static_gas",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SHA3),
				Lt(Gas(), 30),
			),
			Effect: FailEffect(),
		},

		{
			Name: "sha3_with_too_few_elements",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SHA3),
				Lt(StackSize(), 2),
			),
			Effect: FailEffect(),
		},

		{
			Name: "sha3_regular",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SHA3),
				Ge(Gas(), 30),
				Ge(StackSize(), 2),
			),
			Parameter: []Parameter{
				MemoryOffsetParameter{},
				MemorySizeParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Gas -= 30
				s.Pc++
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
			}),
		},
	}...)

	// --- MLOAD ---

	rules = append(rules, []Rule{
		{
			Name: "mload_with_too_little_static_gas",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), MLOAD),
				Lt(Gas(), 3),
			),
			Effect: FailEffect(),
		},

		{
			Name: "mload_with_too_few_elements",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), MLOAD),
				Lt(StackSize(), 1),
			),
			Effect: FailEffect(),
		},

		{
			Name: "mload_regular",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), MLOAD),
				Ge(Gas(), 3),
				Ge(StackSize(), 1),
			),
			Parameter: []Parameter{
				MemoryOffsetParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Gas -= 3
				s.Pc++
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
			}),
		},
	}...)

	// --- MSTORE ---

	rules = append(rules, []Rule{
		{
			Name: "mstore_with_too_little_static_gas",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), MSTORE),
				Lt(Gas(), 3),
			),
			Effect: FailEffect(),
		},

		{
			Name: "mstore_with_too_few_elements",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), MSTORE),
				Lt(StackSize(), 2),
			),
			Effect: FailEffect(),
		},

		{
			Name: "mstore_regular",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), MSTORE),
				Ge(Gas(), 3),
				Ge(StackSize(), 2),
			),
			Parameter: []Parameter{
				MemoryOffsetParameter{},
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Gas -= 3
				s.Pc++
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
			}),
		},
	}...)

	// --- MSTORE8 ---

	rules = append(rules, []Rule{
		{
			Name: "mstore8_with_too_little_static_gas",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), MSTORE8),
				Lt(Gas(), 3),
			),
			Effect: FailEffect(),
		},

		{
			Name: "mstore8_with_too_few_elements",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), MSTORE8),
				Lt(StackSize(), 2),
			),
			Effect: FailEffect(),
		},

		{
			Name: "mstore8_regular",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), MSTORE8),
				Ge(Gas(), 3),
				Ge(StackSize(), 2),
			),
			Parameter: []Parameter{
				MemoryOffsetParameter{},
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Gas -= 3
				s.Pc++
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
			}),
		},
	}...)

	// --- SLOAD ---

	rules = append(rules, []Rule{
		{
			Name: "sload_regular_cold",
			Condition: And(
				RevisionBounds(R09_Berlin, R10_London),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SLOAD),
				Ge(Gas(), 2100),
				Ge(StackSize(), 1),
				IsStorageCold(Param(0)),
			),
			Parameter: []Parameter{
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Gas -= 2100
				s.Pc++
				key := s.Stack.Pop()
				s.Stack.Push(s.Storage.Current[key])
				s.Storage.MarkWarm(key)
			}),
		},

		{
			Name: "sload_with_too_little_gas_cold",
			Condition: And(
				RevisionBounds(R09_Berlin, R10_London),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SLOAD),
				Lt(Gas(), 2100),
				IsStorageCold(Param(0)),
			),
			Parameter: []Parameter{
				NumericParameter{},
			},
			Effect: FailEffect(),
		},

		{
			Name: "sload_regular_warm",
			Condition: And(
				RevisionBounds(R09_Berlin, R10_London),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SLOAD),
				Ge(Gas(), 100),
				Ge(StackSize(), 1),
				IsStorageWarm(Param(0)),
			),
			Parameter: []Parameter{
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Gas -= 100
				s.Pc++
				key := s.Stack.Pop()
				s.Stack.Push(s.Storage.Current[key])
			}),
		},

		{
			Name: "sload_with_too_little_gas_warm",
			Condition: And(
				RevisionBounds(R09_Berlin, R10_London),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SLOAD),
				Lt(Gas(), 100),
				IsStorageWarm(Param(0)),
			),
			Parameter: []Parameter{
				NumericParameter{},
			},
			Effect: FailEffect(),
		},

		{
			Name: "sload_regular_pre_berlin",
			Condition: And(
				RevisionBounds(R07_Istanbul, R07_Istanbul),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SLOAD),
				Ge(Gas(), 800),
				Ge(StackSize(), 1),
			),
			Parameter: []Parameter{
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Gas -= 800
				s.Pc++
				key := s.Stack.Pop()
				s.Stack.Push(s.Storage.Current[key])
			}),
		},

		{
			Name: "sload_with_too_little_gas_pre_berlin",
			Condition: And(
				RevisionBounds(R07_Istanbul, R07_Istanbul),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SLOAD),
				Lt(Gas(), 800),
			),
			Parameter: []Parameter{
				NumericParameter{},
			},
			Effect: FailEffect(),
		},

		{
			Name: "sload_with_too_few_elements",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SLOAD),
				Lt(StackSize(), 1),
			),
			Effect: FailEffect(),
		},
	}...)

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

	rules = append(rules, []Rule{
		{
			Name: "sstore_with_too_little_gas_EIP2200",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SSTORE),
				Lt(Gas(), 2300),
			),
			Effect: FailEffect(),
		},

		{
			Name: "sstore_with_too_few_elements",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), SSTORE),
				Lt(StackSize(), 2),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- JUMP ---

	rules = append(rules, []Rule{
		{
			Name: "jump_with_too_little_gas",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Lt(Gas(), 8),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jump_with_too_few_elements",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Lt(StackSize(), 1),
			),
			Effect: FailEffect(),
		},

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

		{
			Name: "jump_valid_target",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMP),
				Ge(Gas(), 8),
				Ge(StackSize(), 1),
				IsCode(Param(0)),
				Eq(Op(Param(0)), JUMPDEST),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 8
				target := s.Stack.Pop()
				s.Pc = uint16(target.Uint64())
			}),
		},
	}...)

	// --- JUMPI ---

	rules = append(rules, []Rule{
		{
			Name: "jumpi_with_too_little_gas",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Lt(Gas(), 10),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jumpi_with_too_few_elements",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Lt(StackSize(), 2),
			),
			Effect: FailEffect(),
		},

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

		{
			Name: "jumpi_valid_target",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPI),
				Ge(Gas(), 10),
				Ge(StackSize(), 2),
				IsCode(Param(0)),
				Eq(Op(Param(0)), JUMPDEST),
				Ne(Param(1), NewU256(0)),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 10
				target := s.Stack.Pop()
				s.Stack.Pop()
				s.Pc = uint16(target.Uint64())
			}),
		},
	}...)

	// --- JUMPDEST ---

	rules = append(rules, []Rule{
		{
			Name: "jumpdest_with_too_little_gas",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPDEST),
				Lt(Gas(), 1),
			),
			Effect: FailEffect(),
		},

		{
			Name: "jumpdest_regular",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), JUMPDEST),
				Ge(Gas(), 1),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 1
				s.Pc++
			}),
		},
	}...)

	// --- Stack PUSH ---

	for i := 1; i <= 32; i++ {
		rules = append(rules, pushOp(i)...)
	}

	// --- Stack POP ---

	rules = append(rules, []Rule{
		{
			Name: "pop_regular",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), POP),
				Ge(Gas(), 2),
				Ge(StackSize(), 1),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 2
				s.Pc++
				s.Stack.Pop()
			}),
		},

		{
			Name: "pop_with_too_little_gas",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), POP),
				Lt(Gas(), 2),
			),
			Effect: FailEffect(),
		},

		{
			Name: "pop_with_too_few_elements",
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), POP),
				Lt(StackSize(), 1),
			),
			Effect: FailEffect(),
		},
	}...)

	// --- Stack DUP ---

	for i := 1; i <= 16; i++ {
		rules = append(rules, dupOp(i)...)
	}

	// --- Stack SWAP ---

	for i := 1; i <= 16; i++ {
		rules = append(rules, swapOp(i)...)
	}

	// --- End ---

	return NewSpecification(rules...)
}()

func binaryOpWithDynamicCost(
	op OpCode,
	costs uint64,
	effect func(a, b U256) U256,
	dynamicCost func(a, b U256) uint64,
) []Rule {
	name := strings.ToLower(op.String())
	return []Rule{
		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), costs),
				Ge(StackSize(), 2),
			),
			Parameter: []Parameter{
				NumericParameter{},
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Pc++
				a := s.Stack.Pop()
				b := s.Stack.Pop()
				cost := costs + dynamicCost(a, b)
				// TODO: Improve handling of dynamic gas through dedicated constraint.
				if s.Gas < cost {
					s.Status = st.Failed
					s.Gas = 0
					return
				}
				s.Gas = s.Gas - cost
				s.Stack.Push(effect(a, b))
			}),
		},

		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(Gas(), costs),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_too_few_elements", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(StackSize(), 2),
			),
			Effect: FailEffect(),
		},
	}
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
	name := strings.ToLower(op.String())
	return []Rule{
		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), costs),
				Ge(StackSize(), 3),
			),
			Parameter: []Parameter{
				NumericParameter{},
				NumericParameter{},
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Gas = s.Gas - costs
				s.Pc++
				a := s.Stack.Pop()
				b := s.Stack.Pop()
				c := s.Stack.Pop()
				s.Stack.Push(effect(a, b, c))
			}),
		},

		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(Gas(), costs),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_too_few_elements", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(StackSize(), 3),
			),
			Effect: FailEffect(),
		},
	}
}

func unaryOp(
	op OpCode,
	costs uint64,
	effect func(a U256) U256,
) []Rule {
	name := strings.ToLower(op.String())
	return []Rule{
		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), costs),
				Ge(StackSize(), 1),
			),
			Parameter: []Parameter{
				NumericParameter{},
			},
			Effect: Change(func(s *st.State) {
				s.Pc++
				a := s.Stack.Pop()
				s.Gas = s.Gas - costs
				s.Stack.Push(effect(a))
			}),
		},

		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(Gas(), costs),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_too_few_elements", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(StackSize(), 1),
			),
			Effect: FailEffect(),
		},
	}
}

func pushOp(n int) []Rule {
	op := OpCode(int(PUSH1) + n - 1)
	name := strings.ToLower(op.String())
	return []Rule{
		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), 3),
				Lt(StackSize(), st.MaxStackSize),
			),
			Effect: Change(func(s *st.State) {
				s.Gas -= 3
				data := make([]byte, n)
				for i := 0; i < n; i++ {
					b, err := s.Code.GetData(int(s.Pc) + i + 1)
					if err != nil {
						panic(err)
					}
					data[i] = b
				}
				s.Stack.Push(NewU256FromBytes(data...))
				s.Pc += uint16(n) + 1
			}),
		},

		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(Gas(), 3),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_not_enough_space", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(StackSize(), st.MaxStackSize),
			),
			Effect: FailEffect(),
		},
	}
}

func dupOp(n int) []Rule {
	op := OpCode(int(DUP1) + n - 1)
	name := strings.ToLower(op.String())
	return []Rule{
		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), 3),
				Ge(StackSize(), n),
				Lt(StackSize(), st.MaxStackSize),
			),
			Effect: Change(func(s *st.State) {
				s.Pc++
				s.Gas -= 3
				s.Stack.Push(s.Stack.Get(n - 1))
			}),
		},

		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(Gas(), 3),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_too_few_elements", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(StackSize(), n),
				Lt(StackSize(), st.MaxStackSize),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_not_enough_space", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(StackSize(), n),
				Ge(StackSize(), st.MaxStackSize),
			),
			Effect: FailEffect(),
		},
	}
}

func swapOp(n int) []Rule {
	op := OpCode(int(SWAP1) + n - 1)
	name := strings.ToLower(op.String())
	return []Rule{
		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Ge(Gas(), 3),
				Ge(StackSize(), n+1),
			),
			Effect: Change(func(s *st.State) {
				s.Pc++
				s.Gas -= 3
				a := s.Stack.Get(0)
				b := s.Stack.Get(n)
				s.Stack.Set(0, b)
				s.Stack.Set(n, a)
			}),
		},

		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(Gas(), 3),
			),
			Effect: FailEffect(),
		},

		{
			Name: fmt.Sprintf("%v_with_too_few_elements", name),
			Condition: And(
				AnyKnownRevision(),
				Eq(Status(), st.Running),
				Eq(Op(Pc()), op),
				Lt(StackSize(), n+1),
			),
			Effect: FailEffect(),
		},
	}
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

	gasLimit := uint64(2300) // EIP2200
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
			if s.Gas < params.gasCost {
				s.Status = st.Failed
				s.Gas = 0
				return
			}

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
