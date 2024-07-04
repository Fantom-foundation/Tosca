// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package tosca

import (
	"encoding/json"
	"fmt"
	"regexp"
)

func (r Revision) String() string {
	switch r {
	case R07_Istanbul:
		return "Istanbul"
	case R09_Berlin:
		return "Berlin"
	case R10_London:
		return "London"
	case R11_Paris:
		return "Paris"
	case R12_Shanghai:
		return "Shanghai"
	case R13_Cancun:
		return "Cancun"
	case R99_UnknownNextRevision:
		return "UnknownNextRevision"
	default:
		return fmt.Sprintf("Revision(%d)", r)
	}
}

func (r Revision) MarshalJSON() ([]byte, error) {
	revString := r.String()
	reg := regexp.MustCompile(`Revision\([0-9]+\)`)
	if reg.MatchString(revString) {
		return nil, &json.UnsupportedValueError{}
	}
	return json.Marshal(revString)
}

func (r *Revision) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	var revision Revision

	switch s {
	case "Istanbul":
		revision = R07_Istanbul
	case "Berlin":
		revision = R09_Berlin
	case "London":
		revision = R10_London
	case "Paris":
		revision = R11_Paris
	case "Shanghai":
		revision = R12_Shanghai
	case "Cancun":
		revision = R13_Cancun
	case "UnknownNextRevision":
		revision = R99_UnknownNextRevision
	default:
		return &json.InvalidUnmarshalError{}
	}

	*r = revision
	return nil
}
