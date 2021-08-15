package core

import (
	"bytes"
	"io/ioutil"
	"log"
)

type HEAD struct {
	rootDir        string
	Content        string // SHA-1 for a commit or a branch name
	PointsToBranch bool
}

func NewHEAD(rootDir string) *HEAD {
	head := new(HEAD)
	head.rootDir = rootDir
	return head
}
func (head *HEAD) Read() {
	content, err := ioutil.ReadFile(head.rootDir + "/.git/HEAD")
	if err != nil {
		log.Fatal(err)
	}

	branch_name_start_index := bytes.Index(content, []byte("ref:"))
	// if the .git/HEAD file stores a branch name
	if branch_name_start_index != -1 {
		branch_file_path := content[branch_name_start_index+len([]byte("ref:")):]
		branch_file_path = bytes.Trim(branch_file_path, " ")
		branch_file_path = bytes.Trim(branch_file_path, "\n")
		splitted_branch_file_path := bytes.Split(branch_file_path, []byte("/"))
		branch_name := string(splitted_branch_file_path[len(splitted_branch_file_path)-1])
		head.Content = branch_name
		head.PointsToBranch = true
		return
	}

	// if the .git/HEAD file stores the SHA-1 for a commit
	commit_sha1 := bytes.Trim(content, " ")
	commit_sha1 = bytes.Trim(commit_sha1, "\n")
	head.Content = string(commit_sha1)
	head.PointsToBranch = false
}

func (head *HEAD) PointsTo(name string, isBranchName bool) {
	head.Content = name
	head.PointsToBranch = isBranchName

	var content string
	if isBranchName {
		content = "ref: refs/heads/" + head.Content + "\n"
	} else {
		content = head.Content
	}
	err := ioutil.WriteFile(head.rootDir+"/.git/HEAD", []byte(content), 0644)
	if err != nil {
		log.Fatal(err)
	}
}
