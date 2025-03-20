package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/dustin/go-humanize"
)

var (
	lines int
	dir   bool
)

func main() {
	processArgs()
	largest := getByLargest()
	printByLargest(largest)
}

const usage = `Usage of largest:
  -n, --lines int
    	Lines of output (default: 1)
  -d, --dir bool
    	Do not include directories (included by default)
  --debug
    	Debug information
`

func processArgs() {
	flag.IntVar(&lines, "lines", 1, "Lines of output (default: 1).")
	flag.IntVar(&lines, "n", 1, "Lines of output (default: 1).")

	flag.BoolVar(&dir, "d", false, "Include directories.")
	flag.BoolVar(&dir, "dir", false, "Include directories.")

	flag.Usage = func() { fmt.Print(usage) }

	flag.Parse()
}

type Entry struct {
	Name string
	Size uint64
}

func getByLargest() []Entry {
	var target string

	if len(flag.Args()) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			die(err)
		}
		target = cwd

	}

	if len(flag.Args()) == 1 {
		target = flag.Args()[0]
	}

	if len(flag.Args()) > 1 {
		die(errors.New("too many arguments!"))
	}

	files, err := os.ReadDir(target)
	if err != nil {
		die(err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	var entries []Entry
	switch dir {

	case false:
		for _, file := range files {
			stat, err := os.Stat(filepath.Join(target, file.Name()))

			if err != nil {
				die(err)
			}

			if !stat.IsDir() {
				entries = append(entries, Entry{
					Name: file.Name(),
					Size: uint64(stat.Size()),
				})
			}
		}

	case true:
		for _, file := range files {
			stat, err := os.Stat(filepath.Join(target, file.Name()))

			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				die(err)
			}

			if !stat.IsDir() {
				entries = append(entries, Entry{
					Name: file.Name(),
					Size: uint64(stat.Size()),
				})
			} else {
				wg.Add(1)
				go func() {
					size := getDirectorySizeIter(filepath.Join(target, file.Name()))
					mu.Lock()
					entries = append(entries, Entry{
						Name: file.Name(),
						Size: size,
					})
					mu.Unlock()
					wg.Done()
				}()
			}
		}
	}

	wg.Wait()

	return entries
}

func getDirectorySizeIter(root string) uint64 {
	stat, err := os.Stat(root)
	if err != nil {
		return 0
	}

	if !stat.IsDir() {
		if stat.Mode().IsRegular() {
			return uint64(stat.Size())
		}
	}

	var stack []string
	stack = append(stack, root)
	var total uint64

	for len(stack) > 0 {
		dir := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			fullPath := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				stack = append(stack, fullPath)
			} else {
				stat, err := entry.Info()
				if err != nil {
					continue
				}
				if stat.Mode().IsRegular() {
					total += uint64(stat.Size())
				}
			}
		}
	}

	return total
}

// func getDirectorySizeRecur(path string) uint64 {
// 	stat, err := os.Stat(path)
// 	if err != nil {
// 		return 0
// 	}

// 	if !stat.IsDir() {
// 		if stat.Mode().IsRegular() {
// 			return uint64(stat.Size())
// 		}
// 		return 0
// 	}

// 	entries, err := os.ReadDir(path)
// 	if err != nil {
// 		return 0
// 	}

// 	var total uint64
// 	for _, entry := range entries {
// 		fullPath := filepath.Join(path, entry.Name())
// 		size := getDirectorySizeRecur(fullPath)
// 		total += size
// 	}

// 	return total
// }

func printByLargest(largest []Entry) {
	sort.Slice(largest, func(i, j int) bool {
		return largest[i].Size > largest[j].Size
	})

	for i, l := range largest {
		if i >= lines {
			return
		}
		fmt.Printf("%s %s\n", l.Name, humanize.Bytes(l.Size))
	}

}

func die(err error) {
	fmt.Println(err)
	os.Exit(1)
}
