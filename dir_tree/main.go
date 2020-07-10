/*
	Утилита tree.
	Выводит дерево каталогов и файлов (если указана опция -f).
*/

package main

import (
	"fmt"
	"io"
	"os"
	"sort"
)

const emptySymb = rune(' ')
const defaultSymb = rune('│')
const closerSymb = rune('└')
const bridgeSymb = rune('├')

func isFileCloser(index, dirSize int) bool {
	if index == dirSize-1 {
		return true
	}
	return false
}

func printFile(out io.Writer, file os.FileInfo, dirNamePrefix []rune) {
	if dirNamePrefix[len(dirNamePrefix)-1] == defaultSymb {
		dirNamePrefix[len(dirNamePrefix)-1] = bridgeSymb
	}
	for i := 0; i < len(dirNamePrefix); i++ {
		if i != 0 {
			fmt.Fprintf(out, "\t")
		}
		if dirNamePrefix[i] != emptySymb {
			fmt.Fprintf(out, "%c", dirNamePrefix[i])
		}
	}
	if dirNamePrefix[len(dirNamePrefix)-1] == bridgeSymb {
		dirNamePrefix[len(dirNamePrefix)-1] = defaultSymb
	}
	fmt.Fprintf(out, "───%v", file.Name())

	if !file.IsDir() {
		if file.Size() == 0 {
			fmt.Fprintf(out, " (empty)")
		} else {
			fmt.Fprintf(out, " (%vb)", file.Size())
		}
	}
	fmt.Fprintf(out, "\n")
}

func readPath(path string, printFiles bool) ([]os.FileInfo, error) {
	dirFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	files := make([]os.FileInfo, 0)
	for file, err := dirFile.Readdir(1); err != io.EOF; file, err = dirFile.Readdir(1) {
		if err != nil {
			return nil, err
		}
		if file[0].IsDir() || printFiles {
			files = append(files, file[0])
		}
	}
	return files, nil
}

func ConcatenatePaths(first string, second string) string {
	newStr := make([]rune, len(first))
	newStr = []rune(first)
	newStr = append(newStr, '/')
	newStr = append(newStr, []rune(second)...)
	return string(newStr)
}

func printDirectoryFiles(out io.Writer, path string, printFiles bool, dirNamePrefix []rune) error {
	files, err := readPath(path, printFiles) // creates an array of files in this directory
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool { // sorts alphabetically
		return files[i].Name() < files[j].Name()
	})

	for i, file := range files { // printing
		if isFileCloser(i, len(files)) {
			dirNamePrefix[len(dirNamePrefix)-1] = closerSymb
		}
		printFile(out, file, dirNamePrefix)
		if isFileCloser(i, len(files)) {
			dirNamePrefix[len(dirNamePrefix)-1] = emptySymb
		}
		if file.IsDir() {
			tmp := append(dirNamePrefix, defaultSymb)
			err := printDirectoryFiles(out, ConcatenatePaths(path, file.Name()), printFiles, tmp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	dirNamePrefix := []rune{defaultSymb}
	return printDirectoryFiles(out, path, printFiles, dirNamePrefix)
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
