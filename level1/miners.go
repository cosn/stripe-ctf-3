package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const user = "user-hf5hvrhg"
const ledger = "LEDGER.txt"
const diff = "difficulty.txt"
var count int

func main() {
        count = runtime.NumCPU()
	runtime.GOMAXPROCS(count)
	c := make(chan string)
	rand.Seed(time.Now().UnixNano())

	updateLedger()
	difficulty := getDifficulty()
	tree, parent := getGit()

	for i := 0; i < count; i++ {
		go do(c, difficulty, tree, parent)
	}

	for {
		select {
		case h := <-c:
			fmt.Printf("\nFound a candidate: %q\n", h)
			if push() {
				os.Exit(0)
			} else {
				os.Exit(1)
			}
		default:
			fmt.Printf(".")
			time.Sleep(200 * time.Millisecond)
		}
	}

	defer close(c)
}

func do(c chan string, difficulty, tree, parent string) {
	for {
		t := time.Now().Unix()
		body := fmt.Sprintf("tree %v\nparent %v\nauthor CTF user <me@example.com> %v +0000\ncommitter CTF user <me@example.com> %v +0000\nGive me a Gitcoin\n\n%v", tree, parent, t, t, rand.Int())

		sha := sha1.New()
		fmt.Fprintf(sha, "commit %d\x00", len(body))
		sha.Write([]byte(body))
		hash := fmt.Sprintf("%x", sha.Sum(nil))

		if hash < difficulty {
			commit(body, hash)
			c <- hash
			break
		}
	}
}

func push() bool {
	var pushOut bytes.Buffer
	pushCmd := exec.Command("git", "push", "origin", "master", "--force")

	pushCmd.Stderr = &pushOut
	err := pushCmd.Run()
	fmt.Printf("\nPush output: %q\n", pushOut.String())
	if err != nil {
		return false
	}

	return true
}

func commit(body, hash string) {
	fmt.Printf("Minted Gitcoin: %q\n%v\n", hash, body)

	hashCmd := exec.Command("git", "hash-object", "-t", "commit", "--stdin", "-w")
	hashCmd.Stdout = nil
	hashCmd.Stdin = strings.NewReader(body)

	if err := hashCmd.Run(); err != nil {
		panic(err)
	}

	resetCmd := exec.Command("git", "reset", "--hard", hash)
	resetCmd.Stderr = nil

	if err := resetCmd.Run(); err != nil {
		panic(err)
	}
}

func updateLedger() {
	reset()
	f, err := os.OpenFile(ledger, os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(user + ": 1\n"); err != nil {
		panic(err)
	}

	addCmd := exec.Command("git", "add", ledger)
	if err := addCmd.Run(); err != nil {
		panic(err)
	}
}

func reset() {
	fetchCmd := exec.Command("git", "fetch")
	fetchCmd.Stdout = nil

	resetCmd := exec.Command("git", "reset", "--hard", "origin/master")
	resetCmd.Stdout = nil

	if err := fetchCmd.Run(); err != nil {
		panic(err)
	}

	if err := resetCmd.Run(); err != nil {
		panic(err)
	}
}

func getDifficulty() string {
	f, err := ioutil.ReadFile(diff)
	if err != nil {
		panic(err)
	}

	return string(f)
}

func getGit() (string, string) {
	var treeOut, parentOut bytes.Buffer
	treeCmd := exec.Command("git", "write-tree")
	parentCmd := exec.Command("git", "rev-parse", "HEAD")

	treeCmd.Stdout = &treeOut
	parentCmd.Stdout = &parentOut

	if err := treeCmd.Run(); err != nil {
		panic(err)
	}

	if err := parentCmd.Run(); err != nil {
		panic(err)
	}

	return strings.TrimRight(treeOut.String(), "\n"), strings.TrimRight(parentOut.String(), "\n")
}
