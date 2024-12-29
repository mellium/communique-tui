// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package storage_test

import (
	"slices"
	"strconv"
	"testing"

	"mellium.im/communique/internal/storage"
)

var iterTestCases = [...]struct {
	migrations storage.Migrations
	current    uint
	target     uint
	sorted     []uint
	expect     string
}{
	0: {},
	1: {
		migrations: storage.Migrations{
			storage.Migration{Version: 1, Up: "up"},
			storage.Migration{Version: 2, Up: "up"},
			storage.Migration{Version: 0, Up: "up"},
			storage.Migration{Version: 3, Up: "up"},
			storage.Migration{Version: 5, Up: "up"},
		},
		current: 1,
		target:  3,
		sorted:  []uint{2, 3},
		expect:  "up",
	},
	2: {
		migrations: storage.Migrations{
			storage.Migration{Version: 1, Down: "down"},
			storage.Migration{Version: 2, Down: "down"},
			storage.Migration{Version: 3, Down: "down"},
			storage.Migration{Version: 5, Down: "down"},
			storage.Migration{Version: 0, Down: "down"},
		},
		current: 3,
		target:  1,
		sorted:  []uint{3, 2},
		expect:  "down",
	},
	3: {
		migrations: storage.Migrations{
			storage.Migration{Version: 1, Up: "up", Down: "down"},
			storage.Migration{Version: 2, Up: "up", Down: "down"},
		},
		current: 2,
		target:  2,
		sorted:  []uint{},
	},
}

func TestIter(t *testing.T) {
	for i, tc := range iterTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var versions []uint
			for ver, script := range tc.migrations.Run(tc.current, tc.target) {
				versions = append(versions, ver)
				if script != tc.expect {
					t.Fatalf("wrong script returned for migration: want=%v, got=%v", tc.expect, script)
				}
			}
			if slices.Compare(tc.sorted, versions) != 0 {
				t.Fatalf("wrong versions: want=%v, got=%v", tc.sorted, versions)
			}
		})
	}
}
