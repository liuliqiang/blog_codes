package main

type TimeMap interface {
	Set(key, val string, ts int)   // O(logN)
	Get(key string, ts int) string // O(logN)
}

const (
	DefaultArraySize = 16
)

type node struct {
	Val string
	Ts  int
}

type arrayTimeMap struct {
	keyVals map[string][]node
}

func NewArrayTimeMap() TimeMap {
	return &arrayTimeMap{
		keyVals: map[string][]node{}, // order tree, RBTree
	}
}

func (m *arrayTimeMap) Set(key, val string, ts int) {
	vals, ok := m.keyVals[key]
	if !ok {
		vals = make([]node, 0, DefaultArraySize)
	}
	vals = append(vals, node{
		Val: val,
		Ts:  ts,
	})
	m.keyVals[key] = vals
}

func (m *arrayTimeMap) Get(key string, ts int) string {
	vals, ok := m.keyVals[key]
	if !ok {
		return ""
	}

	l, r := 0, len(vals)-1
	for l <= r {
		m := l + (r-l)/2
		if vals[m].Ts > ts {
			r = m - 1
		} else {
			l = m + 1
		}
	}

	/*
	   l, r := -1, len(vals) -1
	   for l < r { // l = 3, r = 4
	       m := l + (r - l) / 2 // m = 3
	       if vals[m] <= ts && l < m{
	           l = m // l = 3
	       } else {
	           r = m - 1
	       }
	   }
	*/
	// compare again here
	if r < 0 {
		return ""
	}

	return vals[r].Val
}

//
