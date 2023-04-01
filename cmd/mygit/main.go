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

	switch command := os.Args[1]; command {
	case "init":
		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/master\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized git directory")

	case "cat-file":
		option := os.Args[2]
		if option == "-p" {
			var (
				blobSHA = os.Args[3]
				dir     = blobSHA[:2]
				file    = blobSHA[2:]
				path    = filepath.Join(".git", "objects", dir, file)
			)
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
			}
			defer f.Close()

			rc, err := zlib.NewReader(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
			}

			splitContent := strings.Split(string(content), "\x00")
			fmt.Print(splitContent[1])
		} else {
			fmt.Fprintf(os.Stderr, "Unknown option %s\n", option)
			os.Exit(1)
		}

	case "hash-object":
		option := os.Args[2]
		if option == "-w" {

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
					header      = append([]byte("blob\x00"+fmt.Sprint(len(content))), []byte{0}...)
					fullContent = append(header, content...)
				)

				return fullContent, nil
			}()
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				return
			}

			compressedContentBytes, err := func() ([]byte, error) {
				var b bytes.Buffer

				w := zlib.NewWriter(&b)
				defer w.Close()

				_, err = w.Write(contentBytes)
				if err != nil {
					return nil, err
				}

				return b.Bytes(), nil
			}()
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				return
			}

			blobSHA, err := func() (string, error) {
				h := sha1.New()

				_, err := h.Write(contentBytes)
				if err != nil {
					return "", err
				}

				blobSHABytes := h.Sum(nil)

				return fmt.Sprintf("%x\n", blobSHABytes), nil
			}()
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				return
			}

			err = func() error {
				var (
					dir  = filepath.Join(".git", "objects", blobSHA[:2])
					file = filepath.Join(dir, blobSHA[2:])
				)
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}

				f, err := os.Create(file)
				if err != nil {
					return err
				}
				defer f.Close()

				_, err = f.Write(compressedContentBytes)
				if err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				return
			}

			fmt.Print(blobSHA)
		} else {
			fmt.Fprintf(os.Stderr, "Unknown option %s\n", option)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}
