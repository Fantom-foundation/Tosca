package cts

import (
	"fmt"
	"strings"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/holiman/uint256"
)

var Specification = func() ct.Specification {

	rules := []ct.Rule{}

	// --- Terminal States ---

	rules = append(rules, []ct.Rule{
		{
			Name:      "stopped_is_end",
			Condition: ct.Eq(ct.Status(), ct.Stopped),
			Effect:    NoEffect(),
		},

		{
			Name:      "returned_is_end",
			Condition: ct.Eq(ct.Status(), ct.Returned),
			Effect:    NoEffect(),
		},

		{
			Name:      "reverted_is_end",
			Condition: ct.Eq(ct.Status(), ct.Reverted),
			Effect:    NoEffect(),
		},

		{
			Name:      "failed_is_end",
			Condition: ct.Eq(ct.Status(), ct.Failed),
			Effect:    NoEffect(),
		},
	}...)

	// --- Invalid Operations ---

	rules = append(rules, []ct.Rule{
		{
			Name: "invalid_operation",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.In(ct.Op(ct.Pc()), getInvalidOps()),
			),
			Effect: Fail(),
		},

		{
			Name: "invalid_pc",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsData(ct.Pc()),
			),
			Effect: Fail(),
		},
	}...)

	// --- STOP ---

	rules = append(rules, ct.Rule{
		Name: "stop_terminates_interpreter",
		Condition: ct.And(
			ct.Eq(ct.Status(), ct.Running),
			ct.IsCode(ct.Pc()),
			ct.Eq(ct.Op(ct.Pc()), ct.STOP),
		),
		Effect: ct.Update(func(s ct.State) ct.State {
			s.Status = ct.Stopped
			return s
		}),
	})

	// --- POP ---

	rules = append(rules, []ct.Rule{
		{
			Name: "pop_with_too_little_gas",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.POP),
				ct.Lt(ct.Gas(), 2),
			),
			Effect: Fail(),
		},
		{
			Name: "pop_with_no_element",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.POP),
				ct.Ge(ct.Gas(), 2),
				ct.Lt(ct.StackSize(), 1),
			),
			Effect: Fail(),
		},
		{
			Name: "pop_regular",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.POP),
				ct.Ge(ct.Gas(), 2),
				ct.Ge(ct.StackSize(), 1),
			),
			Effect: ct.Update(func(s ct.State) ct.State {
				s.Gas = s.Gas - 2
				s.Pc++
				s.Stack.Pop()
				return s
			}),
		},
	}...)

	// --- PUSH1 ---

	rules = append(rules, getPushOpRules(1)...)
	rules = append(rules, getPushOpRules(2)...)
	rules = append(rules, getPushOpRules(16)...)
	rules = append(rules, getPushOpRules(32)...)

	// --- Arithmetic ---

	rules = append(rules, getBinaryOpRules(ct.ADD, 3, func(a, b uint256.Int) uint256.Int {
		return *a.Add(&a, &b)
	})...)

	rules = append(rules, getBinaryOpRules(ct.LT, 3, func(a, b uint256.Int) uint256.Int {
		return boolToUint256(a.Lt(&b))
	})...)

	rules = append(rules, getBinaryOpRules(ct.EQ, 3, func(a, b uint256.Int) uint256.Int {
		return boolToUint256(a.Eq(&b))
	})...)

	// --- MLOAD / MSTORE / MSTORE8
	rules = append(rules, []ct.Rule{
		// MLOAD

		{
			Name: "mload_with_too_little_static_gas",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.MLOAD),
				ct.Lt(ct.Gas(), 3),
			),
			Effect: Fail(),
		},

		{
			Name: "mload_with_too_few_elements",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.MLOAD),
				ct.Ge(ct.Gas(), 3),
				ct.Lt(ct.StackSize(), 1),
			),
			Effect: Fail(),
		},

		{
			Name: "mload_regular",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.MLOAD),
				ct.Ge(ct.Gas(), 3),
				ct.Ge(ct.StackSize(), 1),
			),
			Parameter: []ct.Parameter{
				ct.NumericParameter{},
			},
			Effect: ct.Update(func(s ct.State) ct.State {
				offset_u256 := s.Stack.Pop()
				memCost, offset, _ := s.Memory.ExpansionCosts(&offset_u256, *uint256.NewInt(32))

				if s.Gas < 3+memCost {
					s.Status = ct.Failed
					s.Gas = 0
					return s
				}
				s.Gas -= 3 + memCost

				var value uint256.Int
				value.SetBytes32(s.Memory.ReadFrom(offset, 32))
				s.Stack.Push(value)

				s.Pc++
				return s
			}),
		},

		// MSTORE

		{
			Name: "mstore_with_too_little_static_gas",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.MSTORE),
				ct.Lt(ct.Gas(), 3),
			),
			Effect: Fail(),
		},

		{
			Name: "mstore_with_too_few_elements",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.MSTORE),
				ct.Ge(ct.Gas(), 3),
				ct.Lt(ct.StackSize(), 2),
			),
			Effect: Fail(),
		},

		{
			Name: "mstore_regular",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.MSTORE),
				ct.Ge(ct.Gas(), 3),
				ct.Ge(ct.StackSize(), 2),
			),
			Parameter: []ct.Parameter{
				ct.NumericParameter{},
				ct.NumericParameter{},
			},
			Effect: ct.Update(func(s ct.State) ct.State {
				offset_u256 := s.Stack.Pop()
				value := s.Stack.Pop()
				memCost, offset, _ := s.Memory.ExpansionCosts(&offset_u256, *uint256.NewInt(32))

				if s.Gas < 3+memCost {
					s.Status = ct.Failed
					s.Gas = 0
					return s
				}
				s.Gas -= 3 + memCost

				valueBytes := value.Bytes32()
				s.Memory.WriteTo(valueBytes[:], offset)

				s.Pc++
				return s
			}),
		},

		// MSTORE8

		{
			Name: "mstore8_with_too_little_static_gas",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.MSTORE8),
				ct.Lt(ct.Gas(), 3),
			),
			Effect: Fail(),
		},

		{
			Name: "mstore8_with_too_few_elements",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.MSTORE8),
				ct.Ge(ct.Gas(), 3),
				ct.Lt(ct.StackSize(), 2),
			),
			Effect: Fail(),
		},

		{
			Name: "mstore8_regular",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.MSTORE8),
				ct.Ge(ct.Gas(), 3),
				ct.Ge(ct.StackSize(), 2),
			),
			Parameter: []ct.Parameter{
				ct.NumericParameter{},
				ct.NumericParameter{},
			},
			Effect: ct.Update(func(s ct.State) ct.State {
				offset_u256 := s.Stack.Pop()
				value_u256 := s.Stack.Pop()
				value := value_u256.Bytes32()[31]
				memCost, offset, _ := s.Memory.ExpansionCosts(&offset_u256, *uint256.NewInt(1))

				if s.Gas < 3+memCost {
					s.Status = ct.Failed
					s.Gas = 0
					return s
				}
				s.Gas -= 3 + memCost

				s.Memory.WriteTo([]byte{value}, offset)

				s.Pc++
				return s
			}),
		},
	}...)

	// --- SLOAD / STORE ---

	// SLOAD (with constant gas costs)
	rules = append(rules, []ct.Rule{
		{
			Name: "sload_with_too_little_gas",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.SLOAD),
				ct.Lt(ct.Gas(), 100),
			),
			Effect: Fail(),
		},

		{
			Name: "sload_with_too_few_elements",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.SLOAD),
				ct.Ge(ct.Gas(), 100),
				ct.Lt(ct.StackSize(), 1),
			),
			Effect: Fail(),
		},

		{
			Name: "sload_regular",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.SLOAD),
				ct.Ge(ct.Gas(), 100),
				ct.Ge(ct.StackSize(), 1),
			),
			Parameter: []ct.Parameter{
				ct.NumericParameter{},
			},
			Effect: ct.Update(func(s ct.State) ct.State {
				s.Gas = s.Gas - 100
				s.Pc++
				a := s.Stack.Pop()
				s.Stack.Push(s.Storage.Get(a))
				return s
			}),
		},
	}...)

	// SSTORE (with constant gas costs)
	rules = append(rules, []ct.Rule{
		{
			Name: "sstore_with_too_little_gas",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.SSTORE),
				ct.Lt(ct.Gas(), 100),
			),
			Effect: Fail(),
		},

		{
			Name: "sstore_with_too_few_elements",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.SSTORE),
				ct.Ge(ct.Gas(), 100),
				ct.Lt(ct.StackSize(), 2),
			),
			Effect: Fail(),
		},

		{
			Name: "sstore_regular",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.SSTORE),
				ct.Ge(ct.Gas(), 100),
				ct.Ge(ct.StackSize(), 2),
			),
			Parameter: []ct.Parameter{
				ct.NumericParameter{},
				ct.NumericParameter{},
			},
			Effect: ct.Update(func(s ct.State) ct.State {
				s.Gas = s.Gas - 100
				s.Pc++
				a := s.Stack.Pop()
				b := s.Stack.Pop()
				s.Storage.Set(a, b)
				return s
			}),
		},
	}...)

	// --- JUMP ---

	rules = append(rules, []ct.Rule{
		{
			Name: "jump_with_too_little_gas",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
				ct.Lt(ct.Gas(), 8),
			),
			Effect: Fail(),
		},

		{
			Name: "jump_with_too_few_elements",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
				ct.Ge(ct.Gas(), 8),
				ct.Lt(ct.StackSize(), 1),
			),
			Effect: Fail(),
		},

		{
			Name: "jump_to_data",
			Condition: ct.And(
				ct.Ge(ct.StackSize(), 1),
				ct.IsData(ct.Param(0)),
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
				ct.Ge(ct.Gas(), 8),
			),
			Effect: Fail(),
		},

		{
			Name: "jump_to_invalid_destination",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
				ct.Ge(ct.Gas(), 8),
				ct.Ge(ct.StackSize(), 1),
				ct.IsCode(ct.Param(0)),
				ct.Ne(ct.Op(ct.Param(0)), ct.JUMPDEST),
			),
			Effect: Fail(),
		},

		{
			Name: "jump_valid_target",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMP),
				ct.Ge(ct.Gas(), 8),
				ct.Ge(ct.StackSize(), 1),
				ct.IsCode(ct.Param(0)),
				ct.Eq(ct.Op(ct.Param(0)), ct.JUMPDEST),
			),
			Effect: ct.Update(func(s ct.State) ct.State {
				s.Gas = s.Gas - 8
				target := s.Stack.Pop()
				s.Pc = uint16(target.Uint64())
				return s
			}),
		},
	}...)

	// --- JUMPI ---

	rules = append(rules, []ct.Rule{
		{
			Name: "jumpi_with_too_little_gas",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
				ct.Lt(ct.Gas(), 10),
			),
			Effect: Fail(),
		},

		{
			Name: "jumpi_with_too_few_elements",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
				ct.Ge(ct.Gas(), 10),
				ct.Lt(ct.StackSize(), 2),
			),
			Effect: Fail(),
		},

		{
			Name: "jumpi_not_taken",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
				ct.Ge(ct.Gas(), 10),
				ct.Ge(ct.StackSize(), 2),
				ct.Eq(ct.Param(1), *uint256.NewInt(0)),
			),
			Effect: ct.Update(func(s ct.State) ct.State {
				s.Gas = s.Gas - 10
				s.Stack.Pop()
				s.Stack.Pop()
				s.Pc = s.Pc + 1
				return s
			}),
		},

		{
			Name: "jumpi_to_data",
			Condition: ct.And(
				ct.Ge(ct.StackSize(), 2),
				ct.IsData(ct.Param(0)),
				ct.Ne(ct.Param(1), *uint256.NewInt(0)),
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
				ct.Ge(ct.Gas(), 10),
			),
			Effect: Fail(),
		},

		{
			Name: "jumpi_to_invalid_destination",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
				ct.Ge(ct.Gas(), 10),
				ct.Ge(ct.StackSize(), 2),
				ct.IsCode(ct.Param(0)),
				ct.Ne(ct.Op(ct.Param(0)), ct.JUMPDEST),
				ct.Ne(ct.Param(1), *uint256.NewInt(0)),
			),
			Effect: Fail(),
		},

		{
			Name: "jumpi_valid_target",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMPI),
				ct.Ge(ct.Gas(), 10),
				ct.Ge(ct.StackSize(), 2),
				ct.IsCode(ct.Param(0)),
				ct.Eq(ct.Op(ct.Param(0)), ct.JUMPDEST),
				ct.Ne(ct.Param(1), *uint256.NewInt(0)),
			),
			Effect: ct.Update(func(s ct.State) ct.State {
				s.Gas = s.Gas - 10
				target := s.Stack.Pop()
				s.Stack.Pop()
				s.Pc = uint16(target.Uint64())
				return s
			}),
		},
	}...)

	// --- JUMPDEST ---

	rules = append(rules, []ct.Rule{
		{
			Name: "jumpdest_with_too_little_gas",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMPDEST),
				ct.Lt(ct.Gas(), 1),
			),
			Effect: Fail(),
		},

		{
			Name: "jumpdest_regular",
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), ct.JUMPDEST),
				ct.Ge(ct.Gas(), 1),
			),
			Effect: ct.Update(func(s ct.State) ct.State {
				s.Gas = s.Gas - 1
				s.Pc++
				return s
			}),
		},
	}...)

	return ct.NewSpecification(rules...)
}()

func NoEffect() ct.Effect {
	return ct.Update(func(s ct.State) ct.State { return s })
}

func Fail() ct.Effect {
	return ct.Update(func(s ct.State) ct.State {
		s.Status = ct.Failed
		s.Gas = 0
		return s
	})
}

func getBinaryOpRules(
	op ct.OpCode,
	costs uint64,
	effect func(a, b uint256.Int) uint256.Int,
) []ct.Rule {
	name := strings.ToLower(op.String())
	return []ct.Rule{
		{
			Name: fmt.Sprintf("%v_with_too_little_gas", name),
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), op),
				ct.Lt(ct.Gas(), costs),
			),
			Effect: Fail(),
		},

		{
			Name: fmt.Sprintf("%v_with_too_few_elements", name),
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), op),
				ct.Ge(ct.Gas(), costs),
				ct.Lt(ct.StackSize(), 2),
			),
			Effect: Fail(),
		},

		{
			Name: fmt.Sprintf("%v_regular", name),
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), op),
				ct.Ge(ct.Gas(), costs),
				ct.Ge(ct.StackSize(), 2),
			),
			Parameter: []ct.Parameter{
				ct.NumericParameter{},
				ct.NumericParameter{},
			},
			Effect: ct.Update(func(s ct.State) ct.State {
				s.Gas = s.Gas - costs
				s.Pc++
				a := s.Stack.Pop()
				b := s.Stack.Pop()
				s.Stack.Push(effect(a, b))
				return s
			}),
		},
	}
}

func getPushOpRules(size int) []ct.Rule {
	op := ct.OpCode(int(ct.PUSH1) - 1 + size)
	return []ct.Rule{
		{
			Name: fmt.Sprintf("push%d_with_too_little_gas", size),
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), op),
				ct.Lt(ct.Gas(), 3),
			),
			Effect: Fail(),
		},

		{
			Name: fmt.Sprintf("push%d_with_no_empty_space", size),
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), op),
				ct.Ge(ct.Gas(), 3),
				ct.Ge(ct.StackSize(), 1024),
			),
			Effect: Fail(),
		},

		{
			Name: fmt.Sprintf("push%d_regular", size),
			Condition: ct.And(
				ct.Eq(ct.Status(), ct.Running),
				ct.IsCode(ct.Pc()),
				ct.Eq(ct.Op(ct.Pc()), op),
				ct.Ge(ct.Gas(), 3),
				ct.Lt(ct.StackSize(), 1024),
			),
			Effect: ct.Update(func(s ct.State) ct.State {
				s.Gas = s.Gas - 3
				value := uint256.NewInt(0)
				data := make([]byte, size)
				for i := 0; i < size && int(s.Pc)+1+i < len(s.Code); i++ {
					data[i] = s.Code[int(s.Pc)+1+i]
				}
				value.SetBytes(data)
				s.Stack.Push(*value)
				s.Pc = s.Pc + 1 + uint16(size)
				return s
			}),
		},
	}
}

func getInvalidOps() []ct.OpCode {
	res := make([]ct.OpCode, 0, 256)
	for i := 0; i < 256; i++ {
		op := ct.OpCode(i)
		switch op {
		case ct.STOP,
			ct.ADD, ct.LT, ct.EQ,
			ct.POP, ct.PUSH1, ct.PUSH2, ct.PUSH16, ct.PUSH32,
			ct.JUMP, ct.JUMPI, ct.JUMPDEST,
			ct.MLOAD, ct.MSTORE, ct.MSTORE8,
			ct.SLOAD, ct.SSTORE:
			// skip
		default:
			res = append(res, op)
		}
	}
	return res
}

func boolToUint256(value bool) uint256.Int {
	if value {
		return *uint256.NewInt(1)
	} else {
		return *uint256.NewInt(0)
	}
}
