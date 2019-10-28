package Digraph

func DirectedDFS(d Digraph, s int) {
	marked := []bool{}
	for i := 0; i < d.V(); i++ {
		marked[i] = false
	}
	dfs(d, s, marked)
}

func dfs(d Digraph, s int, marked []bool) {
	marked[s] = true
	for _, e := range d.Adj(s) {
		if !marked[e] {
			dfs(d, e, marked)
		}
	}
}
