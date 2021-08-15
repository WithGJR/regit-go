package core

type Color int

const (
	White = iota
	Gray
	Black
)

type List struct {
	name string
	next *List
}

type GraphNode struct {
	typ      string
	name     string
	sha1Name []byte
	list     *List
	color    Color
}

type Graph struct {
	adjacencyList []GraphNode
	handles       map[string]int
}

func NewGraph() *Graph {
	graph := new(Graph)
	graph.adjacencyList = make([]GraphNode, 0)
	graph.handles = make(map[string]int)
	return graph
}

func (graph *Graph) AddNode(typ string, name string, sha1Name []byte) {
	// Node exists
	if _, ok := graph.handles[name]; ok {
		return
	}

	node := new(GraphNode)
	node.typ = typ
	node.name = name
	node.sha1Name = sha1Name
	node.list = nil

	graph.adjacencyList = append(graph.adjacencyList, *node)
	graph.handles[node.name] = len(graph.adjacencyList) - 1
}

func (graph *Graph) LookUpNode(name string) (*GraphNode, bool) {
	index, ok := graph.handles[name]
	if !ok {
		return nil, false
	}
	return &(graph.adjacencyList[index]), true
}

func (graph *Graph) AddEdge(source string, target string) {
	source_node, ok := graph.LookUpNode(source)
	if !ok {
		return
	}
	_, ok = graph.LookUpNode(target)
	if !ok {
		return
	}

	new_list := new(List)
	new_list.name = target
	new_list.next = nil

	if source_node.list == nil {
		source_node.list = new_list
		return
	}

	list := source_node.list
	for list != nil {
		// Edge exists
		if list.name == target {
			return
		}
		if list.next == nil {
			break
		}
		list = list.next
	}
	list.next = new_list
}

func (graph *Graph) DFS(before_callback func(*GraphNode), after_callback func(*GraphNode)) {
	for i := 0; i < len(graph.adjacencyList); i++ {
		graph.adjacencyList[i].color = White
	}
	for i := 0; i < len(graph.adjacencyList); i++ {
		node := &(graph.adjacencyList[i])
		if node.color == White {
			graph.dfs_visit(node, before_callback, after_callback)
		}
	}
}

func (graph *Graph) dfs_visit(node *GraphNode, before_callback func(*GraphNode), after_callback func(*GraphNode)) {
	node.color = Gray
	before_callback(node)

	list := node.list
	for list != nil {
		target_node, _ := graph.LookUpNode(list.name)

		if target_node.color == White {
			graph.dfs_visit(target_node, before_callback, after_callback)
		}
		list = list.next
	}
	node.color = Black
	after_callback(node)
}
