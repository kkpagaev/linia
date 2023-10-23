package parser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type File struct {
	path    string
	content []byte
}

func get_files(ctx context.Context, path string, prefix string, ch chan File) {
	out, err := exec.Command("find", path, "-type", "f").Output()

	if err != nil {
		fmt.Printf("%s", err)
	}

	output := string(out[:])
	files := strings.Split(output, "\n")

	var wg sync.WaitGroup

	for _, f := range files {
		if !strings.HasSuffix(f, ".ts") {
			continue
		}

		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			content, err := os.ReadFile(f)
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case ch <- File{content: content, path: prefix + strings.Replace(f, path, "", 1)}:
			}
		}(f)
	}
	wg.Wait()
	close(ch)
}
