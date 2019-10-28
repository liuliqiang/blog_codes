package Digraph

type Digraph interface {
	V() int           //顶点总数
	E() int           //边的总数
	AddEdge(v, w int) // 添加一条边 v -> w
	Adj(v int) []int  // 从 v 出发的目标顶点集合
	Reverse() Digraph // 该图的反向图
	String() string   // 对象的字符串表示
}

func NewDigraph() Digraph {
	return &digraph{
		v:   0,
		e:   0,
		adj: [][]int{},
	}
}

type digraph struct {
	v   int
	e   int
	adj [][]int
}

func (d *digraph) V() int {
	return d.v
}

func (d *digraph) E() int {
	return d.e
}

func (d *digraph) AddEdge(v, w int) {
	d.adj[v] = append(d.adj[v], w)
}

func (d *digraph) Adj(v int) []int {
	return d.adj[v]
}

func (d *digraph) Reverse() Digraph {
	rtn := NewDigraph()
	for i := 0; i < d.v; i++ {
		for _, j := range d.Adj(i) {
			rtn.AddEdge(j, i)
		}
	}

	return rtn
}

func (d digraph) String() string {
	panic("implement me")
}
