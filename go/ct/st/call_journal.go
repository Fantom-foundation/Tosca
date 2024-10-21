// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
	"fmt"
	"slices"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// CallJournal is a part of the state modeling the effect of recursive
// contract calls. It covers past calls to verify the proper execution
// of calls as well as the effect of future calls to be triggered
// by CREATE and CALL expressions.
type CallJournal struct {
	Past   []PastCall
	Future []FutureCall
}

func NewCallJournal() *CallJournal {
	return &CallJournal{}
}

func (j *CallJournal) Call(kind tosca.CallKind, parameter tosca.CallParameters) tosca.CallResult {
	// log the call as a past call.
	j.Past = append(j.Past, PastCall{
		Kind:        kind,
		Recipient:   parameter.Recipient,
		Sender:      parameter.Sender,
		Input:       NewBytes(parameter.Input),
		Value:       parameter.Value,
		Gas:         parameter.Gas,
		CodeAddress: parameter.CodeAddress,
	})

	// consume the next future result
	var result FutureCall
	if len(j.Future) > 0 {
		result = j.Future[0]
		j.Future = j.Future[1:]
	}

	gasLeft := parameter.Gas
	if gasLeft < result.GasCosts {
		gasLeft = 0
	} else {
		gasLeft -= result.GasCosts
	}

	return tosca.CallResult{
		Success:        result.Success,
		Output:         result.Output.ToBytes(),
		GasLeft:        gasLeft,
		GasRefund:      result.GasRefund,
		CreatedAddress: result.CreatedAccount,
	}
}

func (j *CallJournal) Equal(other *CallJournal) bool {
	if j == other {
		return true
	}
	equalPast := slices.EqualFunc(j.Past, other.Past, func(a, b PastCall) bool {
		return a.Equal(&b)
	})
	if !equalPast {
		return false
	}
	return slices.EqualFunc(j.Future, other.Future, func(a, b FutureCall) bool {
		return a.Equal(&b)
	})
}

func (j *CallJournal) Diff(other *CallJournal) []string {
	res := []string{}

	if have, got := len(j.Past), len(other.Past); have != got {
		res = append(res, fmt.Sprintf("different length of past calls, %d vs %d", have, got))
	} else {
		for i, call := range j.Past {
			for _, diff := range call.Diff(&other.Past[i]) {
				res = append(res, fmt.Sprintf("Past Call %d: %s", i, diff))
			}
		}
	}

	if have, got := len(j.Future), len(other.Future); have != got {
		res = append(res, fmt.Sprintf("different length of future calls, %d vs %d", have, got))
	} else {
		for i, call := range j.Future {
			for _, diff := range call.Diff(&other.Future[i]) {
				res = append(res, fmt.Sprintf("Future Call %d: %s", i, diff))
			}
		}
	}

	return res
}

func (j *CallJournal) Clone() *CallJournal {
	// PastCall and FutureCall objects can be shallow-cloned
	// since their reference based fields are immutable.
	return &CallJournal{
		Past:   slices.Clone(j.Past),
		Future: slices.Clone(j.Future),
	}
}

// PastCall represents an already processed call. It is part of the state
// model for two reasons to enable the verification of call parameters.
type PastCall struct {
	Kind        tosca.CallKind
	Recipient   tosca.Address
	Sender      tosca.Address
	Input       Bytes
	Value       tosca.Value
	Gas         tosca.Gas
	CodeAddress tosca.Address
}

func (c *PastCall) Equal(other *PastCall) bool {
	return c == other || *c == *other
}

func (c *PastCall) Diff(other *PastCall) []string {
	var res []string
	if have, got := c.Kind, other.Kind; have != got {
		res = append(res, fmt.Sprintf("different call kind: %v vs %v", have, got))
	}
	if have, got := c.Recipient, other.Recipient; have != got {
		res = append(res, fmt.Sprintf("different recipient: %v vs %v", have, got))
	}
	if have, got := c.Sender, other.Sender; have != got {
		res = append(res, fmt.Sprintf("different sender: %v vs %v", have, got))
	}
	if have, got := c.Input, other.Input; have != got {
		res = append(res, fmt.Sprintf("different input: %v vs %v", have, got))
	}
	if have, got := c.Value, other.Value; have != got {
		res = append(res, fmt.Sprintf("different value: %v vs %v", have, got))
	}
	if have, got := c.Gas, other.Gas; have != got {
		res = append(res, fmt.Sprintf("different gas: %v vs %v (diff: %d)", have, got, got-have))
	}
	if have, got := c.CodeAddress, other.CodeAddress; have != got {
		res = append(res, fmt.Sprintf("different code address: %v vs %v", have, got))
	}
	return res
}

type FutureCall struct {
	Success        bool
	Output         Bytes
	GasCosts       tosca.Gas
	GasRefund      tosca.Gas
	CreatedAccount tosca.Address
}

func (c *FutureCall) Equal(other *FutureCall) bool {
	return c == other || *c == *other
}

func (c *FutureCall) Diff(other *FutureCall) []string {
	var res []string
	if have, got := c.Success, other.Success; have != got {
		res = append(res, fmt.Sprintf("different success: %t vs %t", have, got))
	}
	if have, got := c.Output, other.Output; have != got {
		res = append(res, fmt.Sprintf("different output: %v vs %v", have, got))
	}
	if have, got := c.GasCosts, other.GasCosts; have != got {
		res = append(res, fmt.Sprintf("different gas costs: %v vs %v", have, got))
	}
	if have, got := c.GasRefund, other.GasRefund; have != got {
		res = append(res, fmt.Sprintf("different refund: %v vs %v", have, got))
	}
	if have, got := c.CreatedAccount, other.CreatedAccount; have != got {
		res = append(res, fmt.Sprintf("different created account: %v vs %v", have, got))
	}
	return res
}
