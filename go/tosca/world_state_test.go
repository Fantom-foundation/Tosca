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
	"slices"
	"strings"
	"testing"
)

func TestGetAllStorageStatuses(t *testing.T) {
	existing := []StorageStatus{}
	for s := StorageStatus(0); ; s++ {
		if strings.HasPrefix(s.String(), "StorageStatus") {
			break
		}
		existing = append(existing, s)
	}
	all := GetAllStorageStatuses()
	slices.Sort(existing)
	slices.Sort(all)
	if !slices.Equal(existing, all) {
		t.Errorf("Unexpected statuses, wanted: %v vs got: %v", existing, all)
	}
}
