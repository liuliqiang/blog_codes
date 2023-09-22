package main

import (
	"fmt"
	"testing"
)

func TestArrayTimeMap(t *testing.T) {
	for _, tc := range []struct {
		Desc    string
		SetRcds [][]string
		SetTss  []int
		GetRcds []string
		GetTss  []int
		RtnVals []string
	}{
		{
			Desc: "test simple feature",
			SetRcds: [][]string{
				[]string{"k1", "v1"},
				[]string{"k1", "v2"},
				[]string{"k1", "v1"},
			},
			SetTss:  []int{1, 3, 5},
			GetRcds: []string{"k1"},
			GetTss:  []int{1},
			RtnVals: []string{"v1"},
		},
		{
			Desc: "test simple feature with difference keys",
			SetRcds: [][]string{
				[]string{"k1", "v1"},
				[]string{"k1", "v2"},
				[]string{"k2", "v22"},
				[]string{"k1", "v1"},
			},
			SetTss:  []int{1, 3, 3, 5},
			GetRcds: []string{"k1"},
			GetTss:  []int{3},
			RtnVals: []string{"v2"},
		},
		{
			Desc: "test simple feature with 4 diffence vals",
			SetRcds: [][]string{
				[]string{"k1", "v1"},
				[]string{"k1", "v2"},
				[]string{"k1", "v3"},
				[]string{"k1", "v4"},
			},
			SetTss:  []int{1, 2, 3, 4},
			GetRcds: []string{"k1"},
			GetTss:  []int{0, 1, 2, 3, 4, 5},
			RtnVals: []string{"", "v1", "v2", "v3", "v4", "v4"},
		},
		{
			Desc: "test simple feature with 5 diffence vals",
			SetRcds: [][]string{
				[]string{"k1", "v1"},
				[]string{"k1", "v2"},
				[]string{"k1", "v3"},
				[]string{"k1", "v4"},
				[]string{"k1", "v5"},
			},
			SetTss:  []int{1, 2, 3, 4, 5},
			GetRcds: []string{"k1"},
			GetTss:  []int{0, 1, 2, 3, 4, 5, 6},
			RtnVals: []string{"", "v1", "v2", "v3", "v4", "v5", "v5"},
		},
	} {
		tm := NewArrayTimeMap()
		for idx, rcd := range tc.SetRcds {
			tm.Set(rcd[0], rcd[1], tc.SetTss[idx])
		}
		for idx, rcd := range tc.GetRcds {
			fmt.Printf("%d: %s\n", idx, rcd)
			if tm.Get(rcd, tc.GetTss[idx]) != tc.RtnVals[idx] {
				t.Fatalf("%s: tm.Get('%s', %d) should equal to '%s'",
					tc.Desc,
					rcd,
					tc.GetTss[idx],
					tc.RtnVals[idx],
				)
			}
		}
	}
}
