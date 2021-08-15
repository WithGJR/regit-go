package core

import (
	"bytes"
	"strings"
)

// An auxiliary data structure for constructing tree objects based on the content of staging area (index)
type TreeGraph struct {
	graph   *Graph
	objects map[string]*TreeObject
}

func NewTreeGraph() *TreeGraph {
	tg := new(TreeGraph)
	tg.graph = NewGraph()
	tg.graph.AddNode("tree", "/", nil) // root tree
	tg.objects = make(map[string]*TreeObject)
	return tg
}

func (tg *TreeGraph) AddEntry(path string, sha1Name []byte) {
	splitted_path := strings.Split(path, "/")
	if len(splitted_path) == 1 {
		tg.graph.AddNode("blob", path, sha1Name)
		tg.graph.AddEdge("/", path)
		return
	}

	for i := 1; i <= len(splitted_path); i++ {
		sub_path := strings.Join(splitted_path[:i], "/")

		_, ok := tg.graph.LookUpNode(sub_path)
		if !ok {
			if i == len(splitted_path) {
				tg.graph.AddNode("blob", path, sha1Name)
			} else {
				tg.graph.AddNode("tree", sub_path, nil)
			}
		}
		var parent_path string
		if i == 1 {
			parent_path = "/"
		} else {
			parent_path = strings.Join(splitted_path[:i-1], "/")
		}

		tg.graph.AddEdge(parent_path, sub_path)
	}
}

func (tg *TreeGraph) ConstructTreeObjects(rootDir string) []byte {
	tg.graph.DFS(func(node *GraphNode) {
		if node.typ == "tree" {
			tree := NewTreeObject(rootDir)
			tg.objects[node.name] = tree
		}
	}, func(node *GraphNode) {
		if node.typ != "tree" {
			return
		}
		tree := tg.objects[node.name]
		list := node.list
		for list != nil {
			child_node, _ := tg.graph.LookUpNode(list.name)
			var entry bytes.Buffer

			path := strings.Split(child_node.name, "/")
			if child_node.typ == "tree" {
				entry.Write([]byte("040000 " + path[len(path)-1] + "\000"))
			} else {
				entry.Write([]byte("100644 " + path[len(path)-1] + "\000"))
			}
			entry.Write(child_node.sha1Name)
			tree.WriteEntries([][]byte{entry.Bytes()})

			list = list.next
		}
		tree.Obj.WriteToFile()
		node.sha1Name = tree.Obj.HashedFilename
	})

	root_tree, _ := tg.graph.LookUpNode("/")
	return root_tree.sha1Name
}
