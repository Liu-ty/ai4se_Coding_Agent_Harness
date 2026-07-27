package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(2)
	}
	switch os.Args[1] {
	case "streams":
		fmt.Fprintln(os.Stdout, "out")
		fmt.Fprintln(os.Stderr, "err")
		os.Exit(7)
	case "large-output":
		fmt.Fprint(os.Stdout, strings.Repeat("O", 128*1024))
		fmt.Fprint(os.Stderr, strings.Repeat("E", 128*1024))
	case "large-stdout":
		fmt.Fprint(os.Stdout, strings.Repeat("O", 128*1024))
		fmt.Fprint(os.Stderr, "ok")
	case "large-stderr":
		fmt.Fprint(os.Stdout, "ok")
		fmt.Fprint(os.Stderr, strings.Repeat("E", 128*1024))
	case "env":
		for _, entry := range os.Environ() {
			if strings.Contains(entry, "OPENAI_API_KEY") || strings.Contains(entry, "SAFE_EXECUTOR_TEST_VALUE") {
				fmt.Fprintln(os.Stdout, entry)
			}
		}
	case "spawn-child":
		if len(os.Args) != 3 {
			os.Exit(2)
		}
		time.Sleep(100 * time.Millisecond)
		cmd := exec.Command(os.Args[0], "heartbeat", os.Args[2])
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
		fmt.Fprintln(os.Stdout, cmd.Process.Pid)
		time.Sleep(10 * time.Second)
	case "heartbeat":
		if len(os.Args) != 3 {
			os.Exit(2)
		}
		for {
			f, err := os.OpenFile(os.Args[2], os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
			if err == nil {
				_, _ = f.WriteString(".")
				_ = f.Close()
			}
			time.Sleep(40 * time.Millisecond)
		}
	default:
		os.Exit(2)
	}
}
