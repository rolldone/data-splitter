package database

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

// PrintRecentLogTail prints the last `lines` lines from the configured log file
// to stderr. It respects LOG_PATH env var if set; otherwise defaults to
// "logs/data-splitter.log". This is intended to be invoked when a critical
// error happens so the pipeline can see verbose context.
func PrintRecentLogTail(lines int) {
	// Allow override via LOG_PATH; default to logs/data-splitter.log
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "logs/data-splitter.log"
	}

	// Allow configuring lines via LOG_TAIL_LINES env var (fallback to provided `lines`)
	if envLines := os.Getenv("LOG_TAIL_LINES"); envLines != "" {
		if v, err := strconv.Atoi(envLines); err == nil && v > 0 {
			lines = v
		}
	}

	f, err := os.Open(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", logPath, err)
		return
	}
	defer f.Close()

	// Read all lines (file shouldn't be enormous for tail usage) but scan safely.
	scanner := bufio.NewScanner(f)
	var all []string
	for scanner.Scan() {
		all = append(all, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read log file %s: %v\n", logPath, err)
		return
	}

	start := 0
	if len(all) > lines {
		start = len(all) - lines
	}

	fmt.Fprintf(os.Stderr, "--- BEGIN LOG TAIL (%s) last %d lines ---\n", logPath, lines)
	for _, l := range all[start:] {
		fmt.Fprintln(os.Stderr, l)
	}
	fmt.Fprintf(os.Stderr, "---  END LOG TAIL (%s) ---\n", logPath)
}
