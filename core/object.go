package core

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type GitObject struct {
	typ            string
	content        []byte
	rootDir        string
	HashedFilename []byte
}

func (obj *GitObject) WriteToFile() {
	header := []byte(obj.typ + " " + fmt.Sprint(len(obj.content)) + "\000")
	store := make([]byte, len(header)+len(obj.content))
	copy(store, header)
	copy(store[len(header):], obj.content)
	sha1_byte := sha1.Sum(store)
	obj.HashedFilename = sha1_byte[:]
	hashedFilenameStr := hex.EncodeToString(obj.HashedFilename[:])

	var buf bytes.Buffer
	zlibWriter := zlib.NewWriter(&buf)
	zlibWriter.Write(store)
	zlibWriter.Close()
	compressed_content := buf.Bytes()

	prefix_path := obj.rootDir + "/.git/objects/"
	os.Mkdir(prefix_path+hashedFilenameStr[:2], 0755)

	err := ioutil.WriteFile(prefix_path+hashedFilenameStr[:2]+"/"+hashedFilenameStr[2:], compressed_content, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func (obj *GitObject) readFromExistingObject() {
	sha1_name := hex.EncodeToString(obj.HashedFilename)
	content, err := ioutil.ReadFile(obj.rootDir + "/.git/objects/" + sha1_name[:2] + "/" + sha1_name[2:])
	if err != nil {
		log.Fatal(err)
	}

	reader := bytes.NewReader(content)
	decompressed_content_reader, err := zlib.NewReader(reader)
	if err != nil {
		log.Fatal(err)
	}
	decompressed_content, err := io.ReadAll(decompressed_content_reader)
	if err != nil {
		log.Fatal(err)
	}

	header_end_index := bytes.IndexByte(decompressed_content, byte(0))
	if header_end_index == -1 {
		log.Fatal("Error: " + obj.typ + " object '" + sha1_name + "' is broken")
	}
	if header_end_index+1 <= len([]byte(obj.typ+" ")) {
		log.Fatal("Error: " + obj.typ + " object '" + sha1_name + "' header is broken")
	}

	content_size_str := decompressed_content[len([]byte(obj.typ+" ")):header_end_index]
	content_size, err := strconv.Atoi(string(content_size_str))
	if err != nil {
		log.Fatal("Error: " + obj.typ + " object '" + sha1_name + "' header is broken")
	}
	if len(decompressed_content)-(header_end_index+1) != content_size {
		log.Fatal("Error: " + obj.typ + " object '" + sha1_name + "' is broken")
	}

	obj.content = decompressed_content[header_end_index+1:]
}

type BlobObject struct {
	Obj GitObject
}

func NewBlobObject(rootDir string) *BlobObject {
	blob := new(BlobObject)
	blob.Obj.typ = "blob"
	blob.Obj.rootDir = rootDir
	return blob
}

func (blob *BlobObject) CreateFromFile(filename string) {
	content, err := ioutil.ReadFile(blob.Obj.rootDir + "/" + filename)
	if err != nil {
		log.Fatal(err)
	}
	blob.Obj.content = content
}

func (blob *BlobObject) ReadFromExistingObject(sha1Name string) {
	hashed_filename, err := hex.DecodeString(sha1Name)
	if err != nil {
		log.Fatal(err)
	}
	blob.Obj.HashedFilename = hashed_filename
	blob.Obj.readFromExistingObject()
}

type TreeEntry struct {
	FileType       string
	FileName       string
	HashedFilename []byte
}

type TreeObject struct {
	Obj             GitObject
	Entries         []*TreeEntry
	DescendentTrees map[string]*TreeObject
}

func NewTreeObject(rootDir string) *TreeObject {
	tree := new(TreeObject)
	tree.Obj.typ = "tree"
	tree.Obj.rootDir = rootDir
	tree.Entries = make([]*TreeEntry, 0)
	tree.DescendentTrees = make(map[string]*TreeObject)
	return tree
}

func (tree *TreeObject) WriteEntries(entries []*TreeEntry) {
	tree.Entries = append(tree.Entries, entries...)

	var buf bytes.Buffer
	for _, entry := range tree.Entries {
		buf.WriteString(entry.FileType + " " + entry.FileName + "\000")
		buf.Write(entry.HashedFilename)
	}
	tree.Obj.content = buf.Bytes()
}

func (tree *TreeObject) ReadFromExistingObject(sha1Name string) {
	hashed_filename, err := hex.DecodeString(sha1Name)
	if err != nil {
		log.Fatal(err)
	}
	tree.Obj.HashedFilename = hashed_filename
	tree.Obj.readFromExistingObject()

	current_index := 0
	for current_index < len(tree.Obj.content) {
		first_part_end_index := bytes.Index(tree.Obj.content[current_index:], []byte("\000"))
		first_part := tree.Obj.content[current_index : current_index+first_part_end_index]
		second_part := tree.Obj.content[current_index+first_part_end_index+1 : current_index+first_part_end_index+1+20]

		splitted_first_part := bytes.Split(first_part, []byte(" "))
		file_type := string(splitted_first_part[0])

		file_name := splitted_first_part[1]

		entry := new(TreeEntry)
		entry.FileType = file_type
		entry.FileName = string(file_name)
		entry.HashedFilename = second_part
		tree.Entries = append(tree.Entries, entry)

		current_index += first_part_end_index + 20 + 1
	}
}

func (tree *TreeObject) RecursiveRead(sha1Name string) {
	tree.ReadFromExistingObject(sha1Name)
	for _, entry := range tree.Entries {
		file_type, err := strconv.Atoi(entry.FileType)
		if err != nil {
			log.Fatal(err)
		}
		if file_type == 40000 {
			sub_tree := NewTreeObject(tree.Obj.rootDir)
			sub_tree.RecursiveRead(hex.EncodeToString(entry.HashedFilename))
			tree.DescendentTrees[hex.EncodeToString(sub_tree.Obj.HashedFilename)] = sub_tree
			for key, value := range sub_tree.DescendentTrees {
				tree.DescendentTrees[key] = value
			}
		}
	}
}

func (tree *TreeObject) construct_file_path_names(entries []*TreeEntry, root_path string) []string {
	path_names := make([]string, 0)

	for _, entry := range entries {
		file_type, err := strconv.Atoi(entry.FileType)
		if err != nil {
			log.Fatal(err)
		}

		if file_type == 40000 {
			current_entry_tree := tree.DescendentTrees[hex.EncodeToString(entry.HashedFilename)]
			child_path_names := tree.construct_file_path_names(current_entry_tree.Entries, root_path+entry.FileName+"/")
			path_names = append(path_names, child_path_names...)
		} else {
			path_names = append(path_names, root_path+entry.FileName)
		}
	}
	return path_names
}

// return all file path names stored by the tree object
func (tree *TreeObject) FilePathNames() []string {
	return tree.construct_file_path_names(tree.Entries, "")
}

func (tree *TreeObject) construct_files_SHA1(entries []*TreeEntry) [][]byte {
	object_ids := make([][]byte, 0)
	for _, entry := range entries {
		file_type, err := strconv.Atoi(entry.FileType)
		if err != nil {
			log.Fatal(err)
		}

		if file_type == 40000 {
			current_entry_tree := tree.DescendentTrees[hex.EncodeToString(entry.HashedFilename)]
			object_ids = append(object_ids, tree.construct_files_SHA1(current_entry_tree.Entries)...)
		} else {
			object_ids = append(object_ids, entry.HashedFilename)
		}
	}
	return object_ids
}

func (tree *TreeObject) FilesSHA1() [][]byte {
	return tree.construct_files_SHA1(tree.Entries)
}

type CommitObject struct {
	Obj       GitObject
	tree      string
	parents   []string
	author    string
	committer string
	message   string
}

func NewCommitObject(rootDir string) *CommitObject {
	commit := new(CommitObject)
	commit.Obj.typ = "commit"
	commit.Obj.rootDir = rootDir
	commit.parents = make([]string, 0)
	return commit
}

func (commit *CommitObject) ReadFromExistingObject(sha1Name string) {
	hashed_filename, err := hex.DecodeString(sha1Name)
	if err != nil {
		log.Fatal(err)
	}

	commit.Obj.HashedFilename = hashed_filename
	commit.Obj.readFromExistingObject()
	lines := strings.Split(string(commit.Obj.content), "\n")
	if len(lines) < 5 {
		log.Fatal("Error: " + sha1Name + " is not a valid commit object")
	}
	tree_line := lines[0]
	if strings.Index(tree_line, "tree ") != 0 {
		log.Fatal("Error: " + sha1Name + " is not a valid commit object")
	}
	commit.SetTree(tree_line[len("tree "):])

	next_not_parent_index := 1
	parents := make([]string, 0)
	for i := 1; i < len(lines); i++ {
		if !strings.Contains(lines[i], "parent ") {
			next_not_parent_index = i
			break
		}
		parents = append(parents, lines[i][len("parent "):])
	}
	commit.SetParents(parents)

	author_line := lines[next_not_parent_index]
	commit.SetAuthor(author_line[len("author "):])

	committer_line := lines[next_not_parent_index+1]
	commit.SetCommitter(committer_line[len("committer "):])

	// a '\n' character
	if lines[next_not_parent_index+2] != "" {
		log.Fatal("Error: " + sha1Name + " is not a valid commit object")
	}

	message := strings.Join(lines[next_not_parent_index+3:], "\n")
	commit.SetMessage(message)

	commit.GenerateContent()
}

func (commit *CommitObject) SetTree(tree string) {
	commit.tree = tree
}

func (commit *CommitObject) SetParents(parents []string) {
	commit.parents = append(commit.parents, parents...)
}

func (commit *CommitObject) SetAuthor(author string) {
	commit.author = author
}

func (commit *CommitObject) SetCommitter(committer string) {
	commit.committer = committer
}

func (commit *CommitObject) SetMessage(message string) {
	commit.message = message
}

func (commit *CommitObject) GenerateContent() {
	content := make([]byte, 0)
	content = append(content, []byte("tree "+commit.tree+"\n")...)
	for _, parent := range commit.parents {
		content = append(content, []byte("parent "+parent+"\n")...)
	}
	content = append(content, []byte("author "+commit.author+"\n")...)
	content = append(content, []byte("committer "+commit.committer+"\n")...)
	content = append(content, []byte("\n")...)
	content = append(content, []byte(commit.message+"\n")...)
	commit.Obj.content = content
}
