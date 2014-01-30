package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	//"strconv"
	"strings"
)

type Index struct {
	words map[string]bool
	files map[int]string
	idx   map[string][]*loc
	id    int
	path  string
}

type loc struct {
	file, line int
}

const minLength = 3
const words = "words"

func (i *Index) Init(id int) {
	i.words = make(map[string]bool)
	i.files = make(map[int]string)
	i.idx = make(map[string][]*loc)
	i.id = id
	i.path = fmt.Sprintf("idx-%d", id)

	/*
		_, err := i.mkIndexDir("")
		if err != nil {
			log.Fatal("%d: Cannot create base index directory: %v\n", id, err)
		}
	*/
	err := i.loadWords()
	if err != nil {
		log.Fatal("%d: Cannot load words file: %v\n", id, err)
	}
}

/*
func (i *Index) mkIndexDir(word string) (string, error) {
	p := i.getIndexDir(word)
	//log.Printf("%d: Creating index directory: %q", i.id, p)
	return p, os.MkdirAll(p, 0777)
}

func (i *Index) getIndexDir(word string) string {
	p := i.path
	if len(word) > 0 {
		p = path.Join(p, word[0:2], word[0:3], word)
	}

	return p
}*/

func (i *Index) loadWords() error {
	log.Printf("%d: Loading words\n", i.id)

	wf, err := os.OpenFile(words, os.O_RDONLY, 0666)
	defer wf.Close()
	if err != nil {
		return err
	}

	r := bufio.NewReader(wf)
	for {
		l, _, err := r.ReadLine()
		if err == io.EOF {
			break
		}

		i.words[string(l)] = true
	}

	return nil
}

func (i *Index) Index(path string) {
	log.Printf("%d: Indexing %q\n", i.id, path)
	if err := i.indexDir(path, path); err != nil {
		log.Printf("%d: Indexing error: %v\n", i.id, err)
	}

	log.Printf("%d: Done indexing %q\n", i.id, path)
}

func (i *Index) indexDir(root, dir string) (err error) {
	//log.Printf("%d: Indexing directory %q\n", i.id, dir)

	if strings.Contains(dir, ".") {
		log.Printf("%d: Directory %q is hidden, excluding from indexing\n", i.id, dir)
		return
	}

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
	read := bufio.NewReader(f)
	moreLines := true
	for moreLines {
		lines++
		line, err := read.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				moreLines = false
			} else {
				log.Printf("%d: Error reading line in file %q: %s\n", i.id, file, err)
				continue
			}
		}

		words := strings.Split(line, " ")
		for _, word := range words {
			word = strings.TrimRight(word, "\n")

			if wl := len(word); wl < minLength {
				//log.Printf("%d: %q is too short, skipping\n", i.id, word)
				continue
			}

			//fmt.Printf("%d: Word = %q\n", i.id, word)

			for j := 0; j <= len(word); j++ {
				for jj := j + minLength; jj < len(word); jj++ {
					found = false
					w := word[j : jj+1]
					if _, validWord := i.words[w]; !validWord {
						//log.Printf("%d: %q is not a valid word, skipping indexing\n", i.id, w)
						continue
					}

					if int(w[0])%clients != i.id-1 {
						//log.Printf("%d: %q does not belong in this index\n", i.id, w)
						continue
					}

					for _, l := range i.idx[w] {
						if l.file == idx && l.line == lines {
							found = true
							break
						}
					}

					if !found {
						i.idx[w] = append(i.idx[w], &loc{file: idx, line: lines})
					}

					/*
						//fmt.Printf("%d: Indexing %q\n", i.id, w)
						p, err := i.mkIndexDir(w)
						if err != nil {
							log.Printf("%d: Error creating index directory: %v", i.id, err)
							continue
						}

						p = path.Join(p, fmt.Sprintf("%d-%d", idx, lines))
						if _, err := os.Stat(p); err == nil {
							//log.Printf("%d: %q already exists, skipping indexing\n", i.id, p)
							continue
						}

						idxFile, err := os.OpenFile(p, os.O_CREATE, 0666)
						// close directly here since with defer we risk having too many handles open
						idxFile.Close()
						if err != nil {
							log.Printf("%d: Error creating index location file: %v", i.id, err)
							continue
						}
					*/
				}
			}
		}
	}

	return
}

func (i *Index) Search(word string) []string {
	if len(word) < minLength {
		log.Printf("%d: %q does not meet the minimum length requirement of %d\n", i.id, word, minLength)
		return nil
	}

	/*
		p := i.getIndexDir(word)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			log.Printf("%d: %q has not been indexed\n", i.id, p)
			return nil
		}

		d, err := os.Open(p)
		defer d.Close()
		if err != nil {
			log.Printf("%d: Cannot open index directory %q: %v\n", i.id, p, err)
			return nil
		}

		fs, err := d.Readdir(-1)
		if err != nil {
			log.Printf("%d: Cannot read index directory %q: %v\n", i.id, p, err)
			return nil
		}*/
	var res []string
	if loc := i.idx[word]; loc != nil {
		/*
			for _, f := range fs {
				if !f.IsDir() {
					loc := strings.Split(f.Name(), "-")
					file, _ := strconv.Atoi(loc[0])
					line, _ := strconv.Atoi(loc[1])
					res = append(res, fmt.Sprintf("\"%v:%d\"", i.files[file], line))
				}
			}*/
		for _, l := range loc {
			res = append(res, fmt.Sprintf("\"%v:%d\"", i.files[l.file], l.line))
		}
	}

	return res
}
