package main

import (
	"fmt"
	"github.com/alexflint/go-arg"
	"io"
	"os"
	"path/filepath"
)

const (
	dirMode      os.FileMode = 0750
	fileMode     os.FileMode = 0640
	execFileMode os.FileMode = 0750
)

var (
	dirfixerVersion = "0.0.1"
)

func main() {
	var progSignature string
	if len(os.Args) > 0 {
		progSignature = os.Args[0]
	} else {
		progSignature = "dirfixer"
	}

	var args struct {
		FixPath      *string `arg:"positional" help:"path to fix"`
		FailEarly    bool    `arg:"-f,--fail-early" help:"stop iterating over files and folders as soon as an error is encountered"`
		PrintVersion bool    `arg:"-V,--version" help:"print program version"`
	}
	parser := arg.MustParse(&args)

	if args.PrintVersion {
		_, err := fmt.Println("dirfixer version", dirfixerVersion)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s: failed to print version:: %v\n", progSignature, err)
			os.Exit(2)
		}
		os.Exit(0)
	}

	if args.FixPath == nil || *args.FixPath == "" {
		parser.Fail("no fix path specified")
	}

	exists, isDir, err := isValidPath(*args.FixPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: failed to validate path: %v\n", progSignature, err)
		os.Exit(2)
	}

	if !exists {
		_, _ = fmt.Fprintf(os.Stderr, "%s: path does not exist\n", progSignature)
		os.Exit(1)
	}

	if !isDir {
		err = handleFile(*args.FixPath)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s: failed to handle provided file: %v\n", progSignature, err)
			os.Exit(2)
		}
		os.Exit(0)
	}

	err = filepath.Walk(*args.FixPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if args.FailEarly {
				return fmt.Errorf("iterate over path %s: %w", path, err)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "%s: failed to iterate over path %s: %v\n", progSignature, path, err)
			}
		}
		if info.IsDir() {
			err = handleDir(path)
		} else {
			err = handleFile(path)
		}
		if err != nil {
			if args.FailEarly {
				return fmt.Errorf("handle path %s: %w", path, err)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "%s: failed to handle path %s: %v\n", progSignature, path, err)
			}
		}
		return nil
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: failed to fix path %s: %v\n", progSignature, *args.FixPath, err)
		os.Exit(2)
	}

	os.Exit(0)
}

func isValidPath(path string) (bool, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	return true, info.IsDir(), nil
}

func handleFile(path string) error {
	isExe, err := isExecutable(path)
	if err != nil {
		return fmt.Errorf("checking if executable: %w", err)
	}

	if isExe {
		err = os.Chmod(path, execFileMode)
	} else {
		err = os.Chmod(path, fileMode)
	}
	if err != nil {
		return fmt.Errorf("setting file mode: %w", err)
	}

	return nil
}

func handleDir(path string) error {
	err := os.Chmod(path, dirMode)
	if err != nil {
		return fmt.Errorf("changing directory permissions: %w", err)
	}
	return nil
}

func isExecutable(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("opening file: %w", err)
	}

	magicBytes := make([]byte, 4)
	n, err := file.Read(magicBytes)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("reading file: %w", err)
	}

	err = file.Close()
	if err != nil {
		return false, fmt.Errorf("closing file: %w", err)
	}

	// Check for shebang
	if n >= 2 && magicBytes[0] == '#' && magicBytes[1] == '!' {
		return true, nil
	}

	// Check for ELF
	if n == 4 && magicBytes[0] == 0x7F && magicBytes[1] == 'E' && magicBytes[2] == 'L' && magicBytes[3] == 'F' {
		return true, nil
	}

	return false, nil
}
