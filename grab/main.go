package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

/*
*
TODO:

  - A file with a line over 10 MiB still terminates the whole program. Decide
    whether to skip/report it or implement chunk-based matching.

  - Binary files are silently skipped when traversing a directory; direct
    binary-file input is reported as “no match found.” Pick and report a clear
    policy.

  - Check entry only after handling the walk error; it may be nil for an
    inaccessible entry.

  - Process only regular files; device files, FIFOs, and symlink edge cases can
    cause surprising behavior or blocking.

  - lerr exits immediately, so one unreadable file aborts an entire directory
    scan. Typical grep reports the error, continues where possible, and exits 2
    at the end.

  - Features such as -i, -n, -r, fixed-string matching, and non-colour output
    are still absent—but those are feature work, not flaws in the current core.
*/
func main() {
	if len(os.Args) < 3 {
		lerr("Please provide more parameters. Usage: grab [PATTERN] [...FILE|DIRECTORY]")
	}

	pattern := os.Args[1]
	paths := os.Args[2:]

	regEx, err := regexp.Compile(pattern)
	if err != nil {
		lerr("Please provide a valid regex", err)
	}

	foundAnyMatch := false
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			lerr("Could not access path", err)
		}

		var foundMatch bool
		if info.IsDir() {
			foundMatch = parseDir(regEx, path)
		} else {
			foundMatch, _ = parseFile(regEx, path)
		}

		if !foundMatch {
			formattedPath := fmt.Sprintf("%s%s%s:", "\x1b[35m", path, "\x1b[0m")
			_, _ = fmt.Fprintln(os.Stdout, formattedPath, "no match found")
		} else {
			foundAnyMatch = true
		}
	}

	if !foundAnyMatch {
		os.Exit(1)
	}
}

func parseDir(regEx *regexp.Regexp, dirPath string) bool {
	var foundMatch bool
	err := filepath.WalkDir(dirPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			lerr("Could not walk dir", err)
		}

		if entry.IsDir() {
			return nil
		}

		foundMatch, skipped := parseFile(regEx, path)
		if foundMatch || skipped {
			return nil
		}

		return nil
	})
	if err != nil {
		lerr("Error while walking directory", err)
	}

	return foundMatch
}

func parseFile(regEx *regexp.Regexp, path string) (bool, bool) {
	file, err := os.Open(path)
	if err != nil {
		lerr("Could not open file", err)
	}

	defer file.Close()

	isBinary, err := isBinary(file)
	if err != nil {
		lerr("Could not inspect file", err)
	}
	if isBinary {
		return false, true
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(
		make([]byte, 64*1024),
		10*1024*1024,
	)
	foundMatch := false
	for scanner.Scan() {
		line := scanner.Text()

		if regEx.MatchString(line) {
			foundMatch = true
			highlightedLine := regEx.ReplaceAllString(
				line,
				"\x1b[31m$0\x1b[0m",
			)
			formattedPath := fmt.Sprintf("%s%s%s:", "\x1b[35m", path, "\x1b[0m")
			_, _ = fmt.Fprintln(os.Stdout, formattedPath, highlightedLine)
		}
	}

	if err = scanner.Err(); err != nil {
		lerr("Could not read file", err)
	}

	return foundMatch, false
}

func isBinary(file *os.File) (bool, error) {
	const sampleSize = 8 * 1024

	sample := make([]byte, sampleSize)
	n, err := file.Read(sample)
	if err != nil && err != io.EOF {
		return false, err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return false, err
	}

	return bytes.IndexByte(sample[:n], 0) >= 0, nil
}

func lerr(message string, errs ...error) {
	_, _ = fmt.Fprintln(os.Stderr, message)

	for _, err := range errs {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}

	os.Exit(2)
}
