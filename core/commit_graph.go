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
	rootDir        string
}

func NewCommitGraph(root_commit *CommitObject, rootDir string) *CommitGraph {
	cg := new(CommitGraph)
	cg.graph = NewGraph()
	root_commit_sha1 := hex.EncodeToString(root_commit.Obj.HashedFilename)
	cg.graph.AddNode("commit", root_commit_sha1, root_commit.Obj.HashedFilename)
	cg.rootCommitName = root_commit_sha1
	cg.commits = make(map[string]*CommitObject)
	cg.commits[root_commit_sha1] = root_commit
	cg.rootDir = rootDir

	return cg
}

func (cg *CommitGraph) LoadAllCommits() []*CommitObject {
	root_commit_node, _ := cg.graph.LookUpNode(cg.rootCommitName)

	all_commits := make([]*CommitObject, 0)
	cg.graph.BFS(root_commit_node, func(node *GraphNode) {
		current_commit := cg.commits[node.name]
		all_commits = append(all_commits, current_commit)

		for _, parent_sha1 := range current_commit.parents {
			parent_commit := NewCommitObject(cg.rootDir)
			parent_commit.ReadFromExistingObject(parent_sha1)
			cg.commits[parent_sha1] = parent_commit
			cg.graph.AddNode("commit", parent_sha1, parent_commit.Obj.HashedFilename)
			cg.graph.AddEdge(node.name, parent_sha1)
		}

	}, func(node *GraphNode) {
		// nothing to do
	})
	return all_commits
}

func (cg *CommitGraph) PrintCommitLogs() {
	msg_content_builder := new(strings.Builder)
	all_commits := cg.LoadAllCommits()

	for _, commit := range all_commits {
		// print in yellow
		msg_content_builder.WriteString(fmt.Sprintf("\033[33mcommit %s\033[0m\n", hex.EncodeToString(commit.Obj.HashedFilename)))
		msg_content_builder.WriteString(fmt.Sprintln("Author:  " + commit.author))
		msg_content_builder.WriteString(fmt.Sprintln("Committer:  " + commit.committer))
		msg_content_builder.WriteString("\n")
		for _, line := range strings.Split(commit.message, "\n") {
			msg_content_builder.WriteString(fmt.Sprintf("\t%s\n", line))
		}
		msg_content_builder.WriteString("\n")
	}

	// invoke the 'less' command to print the output
	reader := bytes.NewReader([]byte(msg_content_builder.String()))
	less_cmd := exec.Command("less")
	less_cmd.Stdin = reader
	less_cmd.Stdout = os.Stdout
	err := less_cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
