package core

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"sort"
	"syscall"
)

// 12-byte header
type IndexHeader struct {
	signature      []byte //(4-byte)The signature is { 'D', 'I', 'R', 'C' } (stands for "dircache")
	version_number []byte //(4-byte)The current supported versions are 2, 3 and 4.
	entries_count  uint32
}

// Index entries are sorted in ascending order on the name field,
// interpreted as a string of unsigned bytes (i.e. memcmp() order, no
// localization, no special casing of directory separator '/'). Entries
// with the same name are sorted by their stage field.

// An index entry typically represents a file. However, if sparse-checkout
//   is enabled in cone mode (`core.sparseCheckoutCone` is enabled) and the
//   `extensions.sparseIndex` extension is enabled, then the index may
//   contain entries for directories outside of the sparse-checkout definition.
//   These entries have mode `040000`, include the `SKIP_WORKTREE` bit, and
//   the path ends in a directory separator.
type IndexEntry struct {
	// 32-bit ctime seconds, the last time a file's metadata changed
	// this is stat(2) data
	Ctime_sec uint32
	// 32-bit ctime nanosecond fractions
	// this is stat(2) data
	Ctime_nanosec uint32
	// 32-bit mtime seconds, the last time a file's data changed
	// this is stat(2) data
	Mtime_sec uint32
	// 32-bit mtime nanosecond fractions
	// this is stat(2) data
	Mtime_nanosec uint32
	// 32-bit Dev
	// this is stat(2) data
	Dev uint32
	// 32-bit Ino
	// this is stat(2) data
	Ino uint32
	// 32-bit mode, split into (high to low bits):
	// 4-bit object type
	//   valid values in binary are 1000 (regular file), 1010 (symbolic link)
	//   and 1110 (gitlink)

	// 3-bit unused

	// 9-bit unix permission. Only 0755 and 0644 are valid for regular files.
	// Symbolic links and gitlinks have value 0 in this field.
	Mode uint32
	// 32-bit Uid
	// this is stat(2) data
	Uid uint32
	// 32-bit Gid
	// this is stat(2) data
	Gid uint32
	// 32-bit file size
	// This is the on-disk size from stat(2), truncated to 32-bit.
	File_size uint32
	// (SHA-1) Object name for the represented object
	Obj_name []byte
	// A 16-bit 'flags' field split into (high to low bits)

	// 1-bit assume-valid flag

	// 1-bit extended flag (must be zero in version 2)

	// 2-bit stage (during merge)

	// 12-bit name length if the length is less than 0xFFF; otherwise 0xFFF
	// is stored in this field.
	Flags uint16
	// Entry path name (variable length) relative to top level directory
	// (without leading slash). '/' is used as path separator. The special
	// path components ".", ".." and ".git" (without quotes) are disallowed.
	// Trailing slash is also disallowed.

	// The exact encoding is undefined, but the '.' and '/' characters
	// are encoded in 7-bit ASCII and the encoding cannot contain a NUL
	// byte (iow, this is a UNIX pathname).
	Path []byte
}

func (entry *IndexEntry) Stage() uint16 {
	return (entry.Flags >> 12) & 0x03
}

type Index struct {
	header   IndexHeader
	entries  []*IndexEntry
	checksum []byte
	rootDir  string
}

func NewIndex(rootDir string) *Index {
	index := new(Index)
	// default header
	index.header.signature = []byte("DIRC")
	version_num, err := index.transform_number_to_network_bytes(uint32(2))
	if err != nil {
		log.Fatal(err)
	}
	index.header.version_number = version_num
	index.header.entries_count = 0
	index.entries = make([]*IndexEntry, 0)
	index.rootDir = rootDir
	return index
}

func (index *Index) Entries() []*IndexEntry {
	return index.entries
}

func (index *Index) read_number_in_network_byte_order(content []byte, num interface{}) {
	buf := bytes.NewBuffer(content)
	err := binary.Read(buf, binary.BigEndian, num)
	if err != nil {
		log.Fatal(err)
	}
}

func (index *Index) transform_number_to_network_bytes(n interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, n)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func (index *Index) Read() {
	content, err := ioutil.ReadFile(index.rootDir + "/.git/index")
	// index file is empty now
	if err != nil {
		return
	}
	index.header.signature = content[:4]
	if !bytes.Equal(index.header.signature, []byte("DIRC")) {
		log.Fatal("Invalid index signature")
	}

	var version_num uint32
	index.read_number_in_network_byte_order(content[4:8], &version_num)
	if version_num != 2 {
		log.Fatal("Unknown index version")
	}
	index.header.version_number = content[4:8]

	var entries_count uint32
	index.read_number_in_network_byte_order(content[8:12], &entries_count)
	index.header.entries_count = entries_count

	current_index := 12
	for i := 0; i < int(entries_count); i++ {
		entry := new(IndexEntry)
		index.read_number_in_network_byte_order(content[current_index:current_index+4], &(entry.Ctime_sec))
		index.read_number_in_network_byte_order(content[current_index+4:current_index+8], &(entry.Ctime_nanosec))
		index.read_number_in_network_byte_order(content[current_index+8:current_index+12], &(entry.Mtime_sec))
		index.read_number_in_network_byte_order(content[current_index+12:current_index+16], &(entry.Mtime_nanosec))
		index.read_number_in_network_byte_order(content[current_index+16:current_index+20], &(entry.Dev))
		index.read_number_in_network_byte_order(content[current_index+20:current_index+24], &(entry.Ino))
		index.read_number_in_network_byte_order(content[current_index+24:current_index+28], &(entry.Mode))
		index.read_number_in_network_byte_order(content[current_index+28:current_index+32], &(entry.Uid))
		index.read_number_in_network_byte_order(content[current_index+32:current_index+36], &(entry.Gid))
		index.read_number_in_network_byte_order(content[current_index+36:current_index+40], &(entry.File_size))

		entry.Obj_name = content[current_index+40 : current_index+60]
		index.read_number_in_network_byte_order(content[current_index+60:current_index+62], &(entry.Flags))

		buf := bytes.NewBuffer(content[current_index+62:])
		entry.Path, err = buf.ReadBytes(byte(0))
		if err != nil {
			log.Fatal(err)
		}

		trailing_null_byte_count := 0
		// 1-8 nul bytes as necessary to pad the entry to a multiple of eight bytes
		// while keeping the name NUL-terminated.
		// count trailing null bytes
		for next_index := current_index + 62 + len(entry.Path); next_index < len(content) && content[next_index] == byte(0); next_index++ {
			if (62+len(entry.Path)+trailing_null_byte_count)%8 == 0 {
				break
			}
			trailing_null_byte_count++
		}
		current_index = current_index + 62 + len(entry.Path) + trailing_null_byte_count
		index.entries = append(index.entries, entry)
	}
	index.checksum = content[current_index:]
}

func (index *Index) Save() {
	content := make([]byte, 0)
	content = append(content, index.header.signature...)
	content = append(content, index.header.version_number...)

	entries_count, err := index.transform_number_to_network_bytes(index.header.entries_count)
	if err != nil {
		log.Fatal(err)
	}
	content = append(content, entries_count...)

	var i uint32
	for i = 0; i < index.header.entries_count; i++ {
		entry := index.entries[i]
		fields := []string{"Ctime_sec", "Ctime_nanosec", "Mtime_sec", "Mtime_nanosec", "Dev", "Ino", "Mode", "Uid", "Gid", "File_size"}

		for _, field := range fields {
			p := reflect.ValueOf(entry).Elem()

			num, err := index.transform_number_to_network_bytes(p.FieldByName(field).Interface())
			if err != nil {
				log.Fatal(err)
			}
			content = append(content, num...)
		}

		content = append(content, entry.Obj_name...)

		flags, err := index.transform_number_to_network_bytes(entry.Flags)
		if err != nil {
			log.Fatal(err)
		}

		content = append(content, flags...)
		content = append(content, entry.Path...)
		// 1-8 nul bytes as necessary to pad the entry to a multiple of eight bytes
		// while keeping the name NUL-terminated.
		current_total_len := 62 + len(entry.Path)
		if current_total_len%8 != 0 {
			needed_nul_bytes := (current_total_len/8+1)*8 - current_total_len
			nul_bytes := make([]byte, needed_nul_bytes)
			content = append(content, nul_bytes...)
		}
	}

	checksum := sha1.Sum(content)
	content = append(content, checksum[:]...)

	err = ioutil.WriteFile(index.rootDir+"/.git/index", content, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func (index *Index) hasEntry(path_name []byte) int {
	for i, entry := range index.entries {
		if bytes.Equal(entry.Path, path_name) {
			return i
		}
	}
	return -1
}

func (index *Index) sortEntries() {
	keys := make([]string, index.header.entries_count)
	keys_to_index_map := make(map[string]int)
	var i uint32
	for i = 0; i < index.header.entries_count; i++ {
		keys[i] = string(index.entries[i].Path)
		keys_to_index_map[string(index.entries[i].Path)] = int(i)
	}
	sort.Strings(keys)

	sortedEntries := make([]*IndexEntry, index.header.entries_count)
	for i = 0; i < index.header.entries_count; i++ {
		sortedEntries[i] = index.entries[keys_to_index_map[keys[i]]]
	}
	for i = 0; i < index.header.entries_count; i++ {
		index.entries[i] = sortedEntries[i]
	}
}

func (index *Index) WriteEntries(path_names []string, object_ids [][]byte) {
	for i, path := range path_names {
		file, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		fileInfo, err := file.Stat()
		if err != nil {
			log.Fatal(err)
		}
		stat := fileInfo.Sys().(*syscall.Stat_t)
		entry := new(IndexEntry)
		entry.Ctime_sec = uint32(stat.Ctimespec.Sec)
		entry.Ctime_nanosec = uint32(stat.Ctimespec.Nsec)
		entry.Mtime_sec = uint32(stat.Mtimespec.Sec)
		entry.Mtime_nanosec = uint32(stat.Mtimespec.Nsec)
		entry.Dev = uint32(stat.Dev)
		entry.Ino = uint32(stat.Ino)
		entry.Mode = uint32(stat.Mode)
		entry.Uid = stat.Uid
		entry.Gid = stat.Gid
		entry.File_size = uint32(stat.Size)
		entry.Obj_name = object_ids[i]

		if len(path) < 0xfff {
			entry.Flags = uint16(len(path))
		} else {
			entry.Flags = 0xfff
		}

		// path is nul-terminated
		entry.Path = []byte(path + "\000")

		// if the entry is existing
		if i := index.hasEntry(entry.Path); i != -1 {
			index.entries[i] = entry
		} else {
			index.entries = append(index.entries, entry)
		}
	}
	index.header.entries_count = uint32(len(index.entries))
	index.sortEntries()
}

func (index *Index) WriteEmptyStatEntries(path_names []string, object_ids [][]byte, stages []uint16) {
	for i, path_name := range path_names {
		entry := new(IndexEntry)
		entry.Obj_name = object_ids[i]
		entry.Flags = stages[i] << 12
		entry.Path = []byte(path_name + "\000")
		index.entries = append(index.entries, entry)
	}
	index.header.entries_count = uint32(len(index.entries))
	index.sortEntries()
}

func (index *Index) ClearEntries() {
	index.entries = nil
	index.entries = make([]*IndexEntry, 0)
}
