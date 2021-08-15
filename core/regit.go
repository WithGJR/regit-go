package core

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"time"
)

type ReGit struct {
	RootDir string
	Config  map[string]string
}

func NewReGit(rootDir string) *ReGit {
	regit := new(ReGit)
	regit.RootDir = rootDir
	regit.Config = make(map[string]string)
	regit.loadUserConfig()
	return regit
}

func (regit *ReGit) loadUserConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	content, err := ioutil.ReadFile(home + "/.gitconfig")
	if err != nil {
		log.Fatal(err)
	}
	content = bytes.ReplaceAll(content, []byte(" "), []byte(""))
	content = bytes.ReplaceAll(content, []byte("\t"), []byte(""))
	user_section_index := bytes.Index(content, []byte("[user]"))
	if user_section_index == -1 {
		log.Fatal("Error: 'user' section is not present in the .gitconfig file")
	}
	first_char_after_user_section_header_index := user_section_index + len([]byte("[user]"))
	next_section_start_index := bytes.Index(content[first_char_after_user_section_header_index:], []byte("["))

	if next_section_start_index == -1 {
		// read until end
		content = content[first_char_after_user_section_header_index:]
	} else {
		next_section_start_index = first_char_after_user_section_header_index + next_section_start_index
		content = content[first_char_after_user_section_header_index:next_section_start_index]
	}
	content = bytes.Trim(content, "\n")
	lines := bytes.Split(content, []byte("\n"))
	for _, line := range lines {
		key_index := bytes.Index(line, []byte("="))
		if key_index == -1 {
			log.Fatal(".gitconfig file: syntax error")
		}
		regit.Config["user."+string(line[:key_index])] = string(line[key_index+1:])
	}
}

func (regit *ReGit) Init() {
	os.Mkdir(regit.RootDir+"/.git", 0755)
	os.Mkdir(regit.RootDir+"/.git/objects", 0755)
	os.Mkdir(regit.RootDir+"/.git/refs", 0755)
	os.Mkdir(regit.RootDir+"/.git/refs/heads", 0755)
	err := ioutil.WriteFile(regit.RootDir+"/.git/HEAD", []byte("ref: refs/heads/master\n"), 0644)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Initialized empty Git repository in " + regit.RootDir + "/.git")
}

func (regit *ReGit) Add(path_names []string) {
	index := NewIndex(regit.RootDir)
	index.Read()

	blob_path_names := make([]string, 0)
	blob_obj_ids := make([][]byte, 0)
	for _, path := range path_names {
		blob := NewBlobObject(regit.RootDir)
		blob.CreateFromFile(path)
		blob.Obj.WriteToFile()

		blob_path_names = append(blob_path_names, path)
		blob_obj_ids = append(blob_obj_ids, blob.Obj.HashedFilename)
	}
	index.WriteEntries(blob_path_names, blob_obj_ids)
	index.Save()
}

func (regit *ReGit) Commmit(message string) {
	index := NewIndex(regit.RootDir)
	index.Read()

	tg := NewTreeGraph()
	for _, entry := range index.Entries() {
		// path is nul-terminated
		path := entry.Path[:len(entry.Path)-1]
		tg.AddEntry(string(path), entry.Obj_name)
	}

	root_tree_id := tg.ConstructTreeObjects(regit.RootDir)

	now := time.Now()

	commit := NewCommitObject(regit.RootDir)
	commit.SetTree(hex.EncodeToString(root_tree_id[:]))
	commit.SetAuthor(regit.Config["user.name"] + " <" + regit.Config["user.email"] + "> " + strconv.FormatInt(now.Unix(), 10) + " " + now.Format("-0700"))
	commit.SetCommitter(regit.Config["user.name"] + " <" + regit.Config["user.email"] + "> " + strconv.FormatInt(now.Unix(), 10) + " " + now.Format("-0700"))

	head := NewHEAD(regit.RootDir)
	head.Read()
	var branch *Branch
	if head.PointsToBranch {
		branch = NewBranch(head.Content, regit.RootDir)
		branch.Read()

		if len(branch.Commit()) != 0 {
			commit.SetParents([]string{branch.Commit()})
		}
	} else {
		if len(head.Content) != 0 {
			commit.SetParents([]string{head.Content})
		}
	}

	commit.SetMessage(message)
	commit.GenerateContent()
	commit.Obj.WriteToFile()
	if head.PointsToBranch {
		branch.SetCommit(hex.EncodeToString(commit.Obj.HashedFilename))
		branch.Write()
	} else {
		head.PointsTo(hex.EncodeToString(commit.Obj.HashedFilename), false)
	}
	fmt.Println("[commit (" + hex.EncodeToString(commit.Obj.HashedFilename) + ") created] " + message)
}

func (regit *ReGit) Checkout(path_names []string) {
	index := NewIndex(regit.RootDir)
	index.Read()

	path_to_entry_index_map := make(map[string]int)
	entries := index.Entries()
	entry_path_names := make([]string, len(entries))
	for i, entry := range entries {
		entry_path_names[i] = string(entry.Path[:len(entry.Path)-1]) // does not include nul character
	}
	for _, path_name := range path_names {
		entry_index := sort.SearchStrings(entry_path_names, path_name)
		if entry_index == len(entries) {
			log.Fatal("Error: '" + path_name + "' did not match any file(s) known to git")
		}
		path_to_entry_index_map[path_name] = entry_index
	}

	for _, path_name := range path_names {
		entry_index := path_to_entry_index_map[path_name]
		entry := entries[entry_index]
		blob := NewBlobObject(regit.RootDir)
		blob.ReadFromExistingObject(hex.EncodeToString(entry.Obj_name))
		err := ioutil.WriteFile(regit.RootDir+"/"+path_name, blob.Obj.content, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Updated " + strconv.Itoa(len(path_names)) + " path from the index")
}
