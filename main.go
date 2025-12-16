package main

import (
	"log"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

// parseGitLogToTicker converts `git log` output into two long strings:
// one for commit messages and one for "by AUTHOR REL_TIME" meta lines.
func parseGitLogToTicker(logOutput string) (string, string, bool) {
	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	var msgSegs, metaSegs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			continue
		}
		_, author, relTime, subject := parts[0], parts[1], parts[2], parts[3]
		message := subject
		meta := "by " + author + " " + relTime
		// Fixed card width so message/meta line up as columns.
		segmentWidth := len(message)
		if l := len(meta); l > segmentWidth {
			segmentWidth = l
		}
		segmentWidth += 4
		msgSegs = append(msgSegs, padRight(message, segmentWidth))
		metaSegs = append(metaSegs, padRight(meta, segmentWidth))
	}
	if len(msgSegs) == 0 {
		return "", "", false
	}
	return strings.Join(msgSegs, ""), strings.Join(metaSegs, ""), true
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// buildGitTickerText runs git log and returns the scrolling texts.
func buildGitTickerText(maxCommits int) (string, string, bool) {
	cmd := exec.Command("git", "log", "-n", 
		strconv.Itoa(maxCommits), "--pretty=format:%h%x09%an%x09%ar%x09%s")
	out, err := cmd.Output()
	if err != nil {
		return "", "", false
	}
	return parseGitLogToTicker(string(out))
}

func main() {
	rand.Seed(time.Now().UnixNano())

	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("creating screen: %v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("initializing screen: %v", err)
	}
	defer s.Fini()

	s.Clear()
	s.HideCursor()

	width, height := s.Size()
	if width <= 0 || height <= 0 {
		return
	}

	size := width * height
	buffer := make([]int, size+width+1)
	chars := []rune{' ', '.', ':', '^', '*', 'x', 's', 'S', '#', '$'}

	// Colors: dark red -> bright yellow/white.
	styles := []tcell.Style{
		tcell.StyleDefault.Foreground(tcell.ColorBlack),
		tcell.StyleDefault.Foreground(tcell.ColorMaroon),
		tcell.StyleDefault.Foreground(tcell.ColorRed),
		tcell.StyleDefault.Foreground(tcell.ColorDarkOrange),
		tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true),
	}

	msgText, metaText, haveTicker := buildGitTickerText(20)
	msgRow := height - 2
	metaRow := height - 1
	tickerOffset := 0
	frame := 0
	events := make(chan tcell.Event, 10)
	go func() {
		for {
			ev := s.PollEvent()
			if ev == nil {
				return
			}
			events <- ev
		}
	}()

	frameDelay := 30 * time.Millisecond

loop:
	for {
		// Non-blocking input check.
		select {
		case ev := <-events:
			switch ev := ev.(type) {
			case *tcell.EventKey:
				_ = ev // any key exits
				break loop
			case *tcell.EventResize:
				width, height = s.Size()
				if width <= 0 || height <= 0 {
					break loop
				}
				size = width * height
				buffer = make([]int, size+width+1)
				msgRow = height - 2
				metaRow = height - 1
			}
		default:
		}

		// Inject heat on bottom row.
		for i := 0; i < width/9; i++ {
			idx := rand.Intn(width) + width*(height-1)
			if idx >= 0 && idx < len(buffer) {
				buffer[idx] = 65
			}
		}

		// Propagate and cool.
		for i := 0; i < size; i++ {
			b0 := buffer[i]
			b1 := buffer[i+1]
			b2 := buffer[i+width]
			b3 := buffer[i+width+1]
			v := (b0 + b1 + b2 + b3) / 4
			buffer[i] = v
			row := i / width
			col := i % width
			if row >= height || col >= width {
				continue
			}
			// Reserve bottom two lines for git info if available.
			if haveTicker && row >= height-2 {
				continue
			}
			var style tcell.Style
			switch {
			case v > 15:
				style = styles[4]
			case v > 9:
				style = styles[3]
			case v > 4:
				style = styles[2]
			default:
				style = styles[1]
			}
			chIdx := v
			if chIdx > 9 {
				chIdx = 9
			}
			if chIdx < 0 {
				chIdx = 0
			}
			s.SetContent(col, row, chars[chIdx], nil, style)
		}

		// Draw git info as two aligned lines at bottom.
		if haveTicker && height >= 2 && len(msgText) > 0 {
			length := len(msgText)
			if length <= width {
				msgVisible := padRight(msgText, width)
				metaVisible := padRight(metaText, width)
				for x, r := range []rune(msgVisible[:width]) {
					s.SetContent(x, msgRow, r, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
				}
				for x, r := range []rune(metaVisible[:width]) {
					s.SetContent(x, metaRow, r, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
				}
			} else {
				for x := 0; x < width; x++ {
					mi := (tickerOffset + x) % length
					mr := rune(msgText[mi])
					me := rune(metaText[mi])
					s.SetContent(x, msgRow, mr, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
					s.SetContent(x, metaRow, me, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
				}
			}
			if length > 0 && frame%4 == 0 {
				tickerOffset = (tickerOffset + 1) % length
			}
		}

		s.Show()
		time.Sleep(frameDelay)
		frame++
	}
}
