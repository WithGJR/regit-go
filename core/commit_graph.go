package core

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type CommitGraph struct {
	graph          *Graph
	rootCommitName string
	commits        map[string]*CommitObject
}

func NewCommitGraph(root_commit *CommitObject) *CommitGraph {
	cg := new(CommitGraph)
	cg.graph = NewGraph()
	root_commit_sha1 := hex.EncodeToString(root_commit.Obj.HashedFilename)
	cg.graph.AddNode("commit", root_commit_sha1, root_commit.Obj.HashedFilename)
	cg.rootCommitName = root_commit_sha1
	cg.commits = make(map[string]*CommitObject)
	cg.commits[root_commit_sha1] = root_commit

	return cg
}

func (cg *CommitGraph) PrintCommitLogs(rootDir string) {
	root_commit_node, _ := cg.graph.LookUpNode(cg.rootCommitName)

	input_to_less_command := ""
	cg.graph.BFS(root_commit_node, func(node *GraphNode) {
		current_commit := cg.commits[node.name]
		for _, parent_sha1 := range current_commit.parents {
			parent_commit := NewCommitObject(rootDir)
			parent_commit.ReadFromExistingObject(parent_sha1)
			cg.commits[parent_sha1] = parent_commit
			cg.graph.AddNode("commit", parent_sha1, parent_commit.Obj.HashedFilename)
			cg.graph.AddEdge(node.name, parent_sha1)
		}
		// print in yellow
		input_to_less_command += fmt.Sprintf("\033[33mcommit %s\033[0m\n", hex.EncodeToString(current_commit.Obj.HashedFilename))
		input_to_less_command += fmt.Sprintln("Author:  " + current_commit.author)
		input_to_less_command += fmt.Sprintln("Committer:  " + current_commit.committer)
		input_to_less_command += "\n"
		for _, line := range strings.Split(current_commit.message, "\n") {
			input_to_less_command += fmt.Sprintf("\t%s\n", line)
		}
		input_to_less_command += "\n"
	}, func(node *GraphNode) {
		// nothing to do
	})

	// invoke the 'less' command to print the output
	reader := bytes.NewReader([]byte(input_to_less_command))
	less_cmd := exec.Command("less")
	less_cmd.Stdin = reader
	less_cmd.Stdout = os.Stdout
	err := less_cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
