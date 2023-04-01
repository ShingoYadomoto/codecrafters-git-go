package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Usage: your_git.sh <command> <arg1> <arg2> ...
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	var (
		command     = os.Args[1]
		commandFunc func() error
	)
	switch command {
	case "init":
		commandFunc = execInit
	case "cat-file":
		commandFunc = execCatFile
	case "hash-object":
		commandFunc = execHashObject
	case "ls-tree":
		commandFunc = execLsTree
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}

	err := commandFunc()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to exec %s\n", command)
		os.Exit(1)
	}
}

func execInit() error {
	for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	headFileContents := []byte("ref: refs/heads/master\n")
	if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
		return err
	}

	fmt.Println("Initialized git directory")

	return nil
}

func execCatFile() error {
	option := os.Args[2]
	if option != "-p" {
		return fmt.Errorf("Unknown option %s\n", option)
	}

	var (
		blobSHA = os.Args[3]
		dir     = blobSHA[:2]
		file    = blobSHA[2:]
		path    = filepath.Join(".git", "objects", dir, file)
	)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	rc, err := zlib.NewReader(f)
	if err != nil {
		return err
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	splitContent := strings.Split(string(content), "\x00")
	fmt.Print(splitContent[1])

	return nil
}

func execHashObject() error {
	option := os.Args[2]
	if option != "-w" {
		return fmt.Errorf("Unknown option %s\n", option)
	}

	contentBytes, err := func() ([]byte, error) {
		var (
			file = os.Args[3]
		)
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}

		var (
			header      = append([]byte("blob "+fmt.Sprint(len(content))), []byte{0}...)
			fullContent = append(header, content...)
		)

		return fullContent, nil
	}()
	if err != nil {
		return err
	}

	compressedContentBytes, err := func() ([]byte, error) {
		var b bytes.Buffer

		w := zlib.NewWriter(&b)

		_, err = w.Write(contentBytes)
		if err != nil {
			return nil, err
		}
		err = w.Close()
		if err != nil {
			return nil, err
		}

		return b.Bytes(), nil
	}()
	if err != nil {
		return err
	}

	blobSHA, err := func() (string, error) {
		h := sha1.New()

		_, err := h.Write(contentBytes)
		if err != nil {
			return "", err
		}

		blobSHABytes := h.Sum(nil)

		return fmt.Sprintf("%x", blobSHABytes), nil
	}()
	if err != nil {
		return err
	}

	err = func() error {
		var (
			dir  = filepath.Join(".git", "objects", blobSHA[:2])
			file = filepath.Join(dir, blobSHA[2:])
		)

		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}

		err = os.WriteFile(file, compressedContentBytes, 0644)
		if err != nil {
			return err
		}

		return nil
	}()
	if err != nil {
		return err
	}

	fmt.Print(blobSHA)

	return nil
}

func execLsTree() error {
	option := os.Args[2]
	if option != "--name-only" {
		return fmt.Errorf("Unknown option %s\n", option)
	}

	var (
		treeSHA = os.Args[3]
		dir     = treeSHA[:2]
		file    = treeSHA[2:]
		path    = filepath.Join(".git", "objects", dir, file)
	)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	rc, err := zlib.NewReader(f)
	if err != nil {
		return err
	}
	defer rc.Close()

	var contents bytes.Buffer
	_, err = io.Copy(&contents, rc)
	if err != nil {
		return err
	}

	_, err = contents.ReadBytes('\x00')
	if err != nil {
		return err
	}

	for {
		_, err := contents.ReadBytes(' ')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		name, err := contents.ReadBytes('\x00')
		if err != nil {
			return err
		}
		fmt.Print(string(name[:len(name)-1]))

		sha := make([]byte, 20)
		_, err = contents.Read(sha)
		if err != nil {
			return err
		}
		fmt.Println()
	}

	return nil
}
