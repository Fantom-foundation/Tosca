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

// GetStorageStatus obtains the status code to be returned by
// RunContext implementation when mutating a storage slot with
// the given original (=committed), current, and new value.
func GetStorageStatus(original, current, new Word) StorageStatus {
	var zero = Word{}

	// See t.ly/b5HPf for the definition of the return status.
	if current == new {
		return StorageAssigned
	}

	// 0 -> 0 -> Z
	if original == zero && current == zero && new != zero {
		return StorageAdded
	}

	// X -> X -> 0
	if original != zero && current == original && new == zero {
		return StorageDeleted
	}

	// X -> X -> Z
	if original != zero && current == original && new != zero && new != original {
		return StorageModified
	}

	// X -> 0 -> Z
	if original != zero && current == zero && new != original && new != zero {
		return StorageDeletedAdded
	}

	// X -> Y -> 0
	if original != zero && current != original && current != zero && new == zero {
		return StorageModifiedDeleted
	}

	// X -> 0 -> X
	if original != zero && current == zero && new == original {
		return StorageDeletedRestored
	}

	// 0 -> Y -> 0
	if original == zero && current != zero && new == zero {
		return StorageAddedDeleted
	}

	// X -> Y -> X
	if original != zero && current != original && current != zero && new == original {
		return StorageModifiedRestored
	}

	// Default
	return StorageAssigned
}
