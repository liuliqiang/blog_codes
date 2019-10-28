package EdgeWeightedGraph

import "strconv"

type DirectedEdge interface {
	Weight() int
	From() int
	To() int
	String() string
}

type EdgeWeightedDigraph interface {
	V() int
	E() int
	AddEdge(edge DirectedEdge)
	Adj(v int) []DirectedEdge
	Edges() []DirectedEdge
	String() string
}

type directedEdge struct {
	from   int
	to     int
	weight int
}

func (d *directedEdge) Weight() int {
	return d.weight
}

func (d *directedEdge) From() int {
	return d.from
}

func (d *directedEdge) To() int {
	return d.to
}

func (d *directedEdge) String() string {
	return strconv.Itoa(d.from) + " -> " + strconv.Itoa(d.to)
}

type edgeWeightedDigraph struct {
	v   int
	e   int
	adj [][]DirectedEdge
}

func (e *edgeWeightedDigraph) V() int {
	return e.v
}

func (e *edgeWeightedDigraph) E() int {
	return e.e
}

func (e *edgeWeightedDigraph) AddEdge(edge DirectedEdge) {
	e.adj[edge.From()] = append(e.adj[edge.From()], edge)
}

func (e *edgeWeightedDigraph) Adj(v int) []DirectedEdge {
	return e.Adj(v)
}

func (e *edgeWeightedDigraph) Edges() (rtn []DirectedEdge) {
	for v := 0; v < e.V(); v++ {
		for _, e := range e.adj[v] {
			rtn = append(rtn, e)
		}
	}

	return
}

func (e *edgeWeightedDigraph) String() string {
	panic("implement me")
}
