package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type Index struct {
	words map[string]int
	files map[int]string
	idx   map[int]map[string]bool
	id    int
	path  string
}

type loc struct {
	file, line int
}

const minLength = 3
const words = "words"

func (i *Index) Init(id int) {
	i.words = make(map[string]int)
	i.files = make(map[int]string)
	i.idx = make(map[int]map[string]bool)
	i.id = id
	i.path = fmt.Sprintf("idx-%d", id)

	err := i.loadWords()
	if err != nil {
		log.Fatal("%d: Cannot load words file: %v\n", id, err)
	}
}

func (i *Index) loadWords() error {
	//log.Printf("%d: Loading words\n", i.id)
	wf, err := os.OpenFile(words, os.O_RDONLY, 0666)
	defer wf.Close()
	if err != nil {
		return err
	}

	r := bufio.NewReader(wf)
	c := 0
	for {
		l, _, err := r.ReadLine()
		if err == io.EOF {
			break
		}

		if w := string(l); len(l) >= minLength {
			i.words[w] = c
			c++
		}
	}

	return nil
}

func (i *Index) shouldIndex(c string) bool {
	return int(c[0])%clients == i.id-1
}

func (i *Index) Index(path string) {
	log.Printf("%d: Indexing %q\n", i.id, path)
	if err := i.indexDir(path, path); err != nil {
		log.Printf("%d: Indexing error: %v\n", i.id, err)
	}

	log.Printf("%d: Done indexing %q\n", i.id, path)
	log.Printf("%d: Words = %d, Files = %d, Index = %d", i.id, len(i.words), len(i.files), len(i.idx))
}

func (i *Index) indexDir(root, dir string) (err error) {
	//log.Printf("%d: Indexing directory %q\n", i.id, dir)

	d, err := os.Open(dir)
	defer d.Close()
	if err != nil {
		log.Printf("%d: Directory indexing error: %v\n", i.id, err)
		return
	}

	if sd, _ := d.Stat(); sd.IsDir() {
		fs, err := d.Readdir(-1)
		if err != nil {
			log.Printf("%d: Directory read error: %v\n", i.id, err)
			return err
		}

		for _, f := range fs {
			if f.IsDir() {
				err = i.indexDir(root, path.Join(dir, f.Name()))
			} else if f.Mode().IsRegular() {
				err = i.indexFile(root, path.Join(dir, f.Name()))
			}
		}
	} else {
		err = i.indexFile(root, d.Name())
	}

	//log.Printf("%d: Done indexing directory %q\n", i.id, dir)
	return
}

func (i *Index) indexFile(root, file string) (err error) {
	//log.Printf("%d: Indexing file %q\n", i.id, file)

	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		log.Printf("%d: File indexing error: %v\n", i.id, err)
		return
	}

	idx := 0
	found := false
	file, _ = filepath.Rel(root, file)

	for k, v := range i.files {
		if file == v {
			idx = k
			found = true
			break
		}
	}

	if !found {
		idx = len(i.files)
		i.files[idx] = file
	}

	var lines int
	lScanner := bufio.NewScanner(f)
	lScanner.Split(bufio.ScanLines)
	for lScanner.Scan() {
		lines++
		key := fmt.Sprintf("%d:%d", idx, lines)
		wScanner := bufio.NewScanner(strings.NewReader(lScanner.Text()))
		wScanner.Split(bufio.ScanWords)
		for wScanner.Scan() {
			word := wScanner.Text()
			word = strings.TrimSuffix(word, ".")

			if len(word) < minLength {
				//log.Printf("%d: %q is too short, skipping\n", i.id, word)
				continue
			}

			if _, valid := i.words[word]; !valid {
				continue
			}

			for j := 0; j <= len(word); j++ {
				for jj := j + minLength; jj < len(word); jj++ {
					w := word[j : jj+1]

					var wp int
					wp, valid := i.words[w]
					if !valid {
						//log.Printf("%d: %q is not a valid word, skipping indexing\n", i.id, w)
						continue
					}

					if !i.shouldIndex(w) {
						//log.Printf("%d: %q does not belong in this index: %d\n", i.id, w, int(w[0])%(clients+1))
						continue
					}

					if _, found = i.idx[wp][key]; !found {
						add(i.idx, wp, key)
					}
				}
			}
		}
	}

	return
}

func add(m map[int]map[string]bool, word int, key string) {
	mm, ok := m[word]
	if !ok {
		mm = make(map[string]bool)
		m[word] = mm
	}
	mm[key] = true
}

func (i *Index) Search(word string) []string {
	if len(word) < minLength {
		log.Printf("%d: %q does not meet the minimum length requirement of %d\n", i.id, word, minLength)
		return nil
	}

	var res []string
	if wp, ok := i.words[word]; ok {
		if e := i.idx[wp]; e != nil {
			for k, _ := range e {
				loc := strings.Split(k, ":")
				file, _ := strconv.Atoi(loc[0])
				res = append(res, fmt.Sprintf("\"%v:%v\"", i.files[file], loc[1]))
			}
		}
	}

	return res
}
