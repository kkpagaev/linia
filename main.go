package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/smoke-laboratory/tools/linia/parser"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <path> <prefix> <output_file_path>")
		return
	}
	ctx := context.Background()
	timeout_ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	res := parser.Run(timeout_ctx, os.Args[1], os.Args[2])
	filePath := os.Args[3]

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error opening the file:", err)
		return
	}
	defer file.Close()

	data := []byte(res)
	_, err = file.Write(data)
	if err != nil {
		fmt.Println("Error writing to the file:", err)
		return
	}

	fmt.Println("Data has been written to the file:", filePath)
}
