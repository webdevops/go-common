package terminal

import (
	"fmt"
	"io"
	"os"
)

// Println prints line on stdout
func Println(msg string, args ...any) {
	Fprintln(os.Stdout, msg, args...)
}

// Printf prints content on stdout
func Printf(msg string, args ...any) {
	Fprintf(os.Stdout, msg, args...)
}

// Fprintln prints line on writer
func Fprintln(w io.Writer, msg string, args ...any) {
	Sync()
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	if _, err := fmt.Fprintln(w, msg); err != nil {
		panic(err)
	}

	Sync()
}

// Fprintf writes content on writer
func Fprintf(w io.Writer, msg string, args ...any) {
	Sync()

	if _, err := fmt.Fprintf(w, msg, args...); err != nil {
		panic(err)
	}

	Sync()
}

// SyncBlock syncs stdout and stderr and calls callback in between sync calls
func SyncBlock(callback func()) {
	Sync()
	callback()
	Sync()
}

// Sync syncs terminal output and ensures logger has finished
func Sync() {
	if err := os.Stdout.Sync(); err != nil {
		panic(err)
	}
	if err := os.Stderr.Sync(); err != nil {
		panic(err)
	}
}
