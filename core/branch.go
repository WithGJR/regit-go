package core

import (
	"fmt"
	"io/ioutil"
	"log"
)

type Branch struct {
	name       string
	commitSHA1 string
	rootDir    string
}

func NewBranch(name string, rootDir string) *Branch {
	branch := new(Branch)
	branch.name = name
	branch.rootDir = rootDir
	return branch
}

func (branch *Branch) Read() {
	content, err := ioutil.ReadFile(branch.rootDir + "/.git/refs/heads/" + branch.name)
	if err != nil {
		return
	}

	branch.commitSHA1 = string(content[:len(content)-1])
}

func (branch *Branch) Commit() string {
	return branch.commitSHA1
}

func (branch *Branch) SetCommit(commit string) {
	branch.commitSHA1 = commit
}

func (branch *Branch) Write() {
	if branch.commitSHA1 == "" {
		fmt.Printf("Error: branch can not be created without any commit")
		return
	}
	err := ioutil.WriteFile(branch.rootDir+"/.git/refs/heads/"+branch.name, []byte(branch.commitSHA1+"\n"), 0644)
	if err != nil {
		log.Fatal(err)
	}
}
