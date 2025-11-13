package main

import (
	"bufio"
	"errors"
	"fmt"
	"golang.org/x/term"
	"os"
	"regexp"
	"time"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)
const DEFAULT_H = 30
const DEFAULT_W = 30
var CELL_W, CELL_H = 8.0, 16.0

type FeedieClientAction int
const(
	main_graphical FeedieClientAction = iota
	add_feed
)

var configPath string

func parseArgs() (FeedieClientAction, *string){
	if home, exists := os.LookupEnv("HOME"); exists{
		configPath = fmt.Sprintf("%s/.config/feedie/conf.json", home)

	} else{
		log.Fatal("You don't have a set $HOME variable? What the hell are you running?")
	}
	for i, arg := range os.Args{
		switch(arg){
		case "-c", "--config":
			if i+1 >= len(os.Args) {log.Fatal(errors.New("Expected value for --config"))}
			configPath = os.Args[i+1]
			continue
		case "--add_feed":
			if i+1 >= len(os.Args) {log.Fatal(errors.New("Expected value for --add_feed"))}
			return add_feed, &os.Args[i+1]
		}
	}
	return main_graphical, nil
}

func main() {
	action, feed := parseArgs()
	config := parseConfigFile(configPath)
	switch (action){
	case main_graphical:
		_ = feed
		// this is here because I don't want it to potentially mess up the
		// image drawing, needing to access stdin/stdout
		if config.getThumbnailBackend() == ueberzug{
			var err error
			CELL_W, CELL_H, err = GetTerminalCellSize()
			if err != nil{
				CELL_W = 8.0
				CELL_H = 16.0
			}

		}

		p := tea.NewProgram(initialSelectModel(getSelectOptions, config), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}
	case add_feed:
		getActionFunc(addFeed_t)(config, []string{*feed})
	}
}

func GetTerminalCellSize() (cellW, cellH float64, err error) {
	stdinFD := int(os.Stdin.Fd())
	stdout := os.Stdout

	// Put stdin into raw mode so we can read immediate escape-sequence replies.
	oldState, err := term.MakeRaw(stdinFD)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to enable raw mode: %w", err)
	}
	defer func() { _ = term.Restore(stdinFD, oldState) }()

	// Use a small helper to write a query and read back one CSI ... t reply.
	readReply := func(query string, timeout time.Duration) (string, error) {
		if _, err := fmt.Fprint(stdout, query); err != nil {
			return "", fmt.Errorf("write failed: %w", err)
		}

		// Read until we see a trailing 't' (end of the DECRQSS/CSI report) or timeout.
		r := bufio.NewReader(os.Stdin)
		var buf []byte
		deadline := time.Now().Add(timeout)

		for {
			// Handle timeout manually; Stdin often lacks SetReadDeadline portability.
			if time.Now().After(deadline) {
				return "", errors.New("timeout waiting for terminal reply")
			}
			_ = os.Stdin.SetReadDeadline(deadline) // Best effort; ignored on some OSes.

			b, err := r.ReadByte()
			if err != nil {
				// brief sleep to avoid busy loop if deadline isn't respected
				time.Sleep(5 * time.Millisecond)
				continue
			}
			buf = append(buf, b)
			if b == 't' {
				// We *might* have read more than just the target reply; keep going a bit
				// to collect contiguous bytes, then break. This is conservative.
				r.Peek(r.Buffered())
				break
			}
			// Safety cap to avoid unbounded buffer growth if terminal is noisy.
			if len(buf) > 4096 {
				return "", errors.New("unexpectedly large reply from terminal")
			}
		}
		return string(buf), nil
	}

	// Regex to capture CSI [optional ?] <kind> ; <h> ; <w> t
	// e.g. "\x1b[4;768;1024t"  OR  "\x1b[8;60;120t"
	re := regexp.MustCompile(`\x1b\[\??(\d+);(\d+);(\d+)t`)

	// 1) Ask for window size in *pixels*: CSI 14 t -> reply kind=4
	pixReply, err := readReply("\x1b[14t", 300*time.Millisecond)
	if err != nil {
		return 0, 0, fmt.Errorf("pixel-size query failed: %w", err)
	}

	// 2) Ask for window size in *cells*: CSI 18 t -> reply kind=8
	cellReply, err := readReply("\x1b[18t", 300*time.Millisecond)
	if err != nil {
		return 0, 0, fmt.Errorf("cell-size query failed: %w", err)
	}

	// Parse replies
	parse := func(s string, wantKind string) (w, h int, e error) {
		matches := re.FindAllStringSubmatch(s, -1)
		for _, m := range matches {
			if len(m) != 4 {
				continue
			}
			kind := m[1]
			hStr := m[2]
			wStr := m[3]
			var hh, ww int
			_, _ = fmt.Sscanf(hStr, "%d", &hh)
			_, _ = fmt.Sscanf(wStr, "%d", &ww)
			if kind == wantKind && hh > 0 && ww > 0 {
				return ww, hh, nil
			}
		}
		return 0, 0, fmt.Errorf("no matching kind=%s reply in %q", wantKind, s)
	}

	winPxW, winPxH, err := func() (int, int, error) { // kind=4 => pixels
		w, h, e := parse(pixReply, "4")
		return w, h, e
	}()
	if err != nil {
		return 0, 0, fmt.Errorf("parse pixel reply: %w", err)
	}

	winCols, winRows, err := func() (int, int, error) { // kind=8 => cells
		w, h, e := parse(cellReply, "8")
		return w, h, e
	}()
	if err != nil {
		return 0, 0, fmt.Errorf("parse cell reply: %w", err)
	}

	if winCols == 0 || winRows == 0 {
		return 0, 0, errors.New("terminal reported zero columns/rows")
	}

	return float64(winPxW) / float64(winCols), float64(winPxH) / float64(winRows), nil
}
