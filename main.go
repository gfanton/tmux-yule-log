package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/peterbourgon/ff/v3/ffcli"

	"yule-log/internal/fire"
	"yule-log/internal/lock"
)

// ---- Constants

const (
	// Timing
	frameDelay         = 30 * time.Millisecond
	defaultIdleTimeout = 300
	pollInterval       = 5

	// Fire simulation
	maxTickerCommits  = 20
	defaultHeatPower  = 75
	heatSourceDivisor = 6
	minHeat           = 10
	maxHeat           = 85
	minSources        = 1
)

// Mode represents the screensaver operating mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModePlayground
	ModeLock
)

// ---- Visual Themes

type theme struct {
	chars  []rune
	styles []tcell.Style
}

var (
	fireTheme = theme{
		chars: []rune{' ', '.', ':', '^', '*', 'x', 's', 'S', '#', '$'},
		styles: []tcell.Style{
			tcell.StyleDefault.Foreground(tcell.ColorBlack),
			tcell.StyleDefault.Foreground(tcell.ColorMaroon),
			tcell.StyleDefault.Foreground(tcell.ColorRed),
			tcell.StyleDefault.Foreground(tcell.ColorDarkOrange),
			tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true),
		},
	}

	contribTheme = theme{
		chars: []rune{' ', '⬝', '⬝', '⯀', '⯀', '◼', '◼', '■', '■', '■'},
		styles: []tcell.Style{
			tcell.StyleDefault.Foreground(tcell.ColorBlack),
			tcell.StyleDefault.Foreground(tcell.NewRGBColor(155, 233, 168)),
			tcell.StyleDefault.Foreground(tcell.NewRGBColor(64, 196, 99)),
			tcell.StyleDefault.Foreground(tcell.NewRGBColor(48, 161, 78)),
			tcell.StyleDefault.Foreground(tcell.NewRGBColor(33, 110, 57)),
		},
	}
)

// ---- Screensaver Configuration & State

type screensaverConfig struct {
	mode     Mode
	contribs bool
	gitDir   string
	noTicker bool
	cooldown fire.CooldownSpeed
}

func (c screensaverConfig) theme() theme {
	if c.contribs {
		return contribTheme
	}
	return fireTheme
}

func (c screensaverConfig) usesVisualState() bool {
	return c.mode == ModePlayground || c.mode == ModeLock
}

type screensaver struct {
	cfg    screensaverConfig
	screen tcell.Screen
	theme  theme

	// Dimensions
	width, height int

	// Fire state
	buffer      []int
	heatPower   int
	heatSources int

	// Ticker state
	msgText, metaText string
	haveTicker        bool
	tickerOffset      int
	frame             int

	// Interactive state (nil in normal mode)
	visualState *fire.VisualState
	inputBuffer *lock.SecureBuffer

	// Event channel
	events chan tcell.Event
}

func newScreensaver(cfg screensaverConfig) (*screensaver, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("creating screen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("initializing screen: %w", err)
	}

	s := &screensaver{
		cfg:       cfg,
		screen:    screen,
		theme:     cfg.theme(),
		heatPower: defaultHeatPower,
		events:    make(chan tcell.Event, 10),
	}

	if cfg.usesVisualState() {
		s.visualState = fire.NewVisualStateWithPreset(cfg.cooldown)
		s.heatPower = s.visualState.EffectiveHeatPower()
	}

	if cfg.mode == ModeLock {
		s.inputBuffer = lock.NewSecureBuffer()
	}

	s.resize()
	s.loadTicker()

	return s, nil
}

func (s *screensaver) close() {
	if s.inputBuffer != nil {
		s.inputBuffer.Destroy()
	}
	s.screen.Fini()
}

func (s *screensaver) resize() {
	s.width, s.height = s.screen.Size()
	if s.width <= 0 || s.height <= 0 {
		return
	}
	size := s.width * s.height
	// Extra space (width+1) for fire propagation lookups: i+1, i+width, i+width+1
	s.buffer = make([]int, size+s.width+1)
	s.heatSources = s.width / heatSourceDivisor
}

func (s *screensaver) loadTicker() {
	if s.cfg.noTicker {
		return
	}
	s.msgText, s.metaText, s.haveTicker = buildGitTickerText(maxTickerCommits, s.cfg.gitDir)
}

// ---- Event Handling

type action int

const (
	actionNone action = iota
	actionExit
	actionResize
)

func (s *screensaver) handleEvent(ev tcell.Event) action {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		s.resize()
		if s.width <= 0 || s.height <= 0 {
			return actionExit
		}
		return actionResize

	case *tcell.EventKey:
		return s.handleKey(ev)
	}
	return actionNone
}

func (s *screensaver) handleKey(ev *tcell.EventKey) action {
	// Feed fire in interactive modes
	if s.visualState != nil {
		s.visualState.OnKeyPress()
		s.heatPower = s.visualState.EffectiveHeatPower()
	}

	switch s.cfg.mode {
	case ModeLock:
		return s.handleKeyLock(ev)
	case ModePlayground:
		return s.handleKeyPlayground(ev)
	default:
		return s.handleKeyNormal(ev)
	}
}

func (s *screensaver) handleKeyNormal(ev *tcell.EventKey) action {
	switch ev.Key() {
	case tcell.KeyEscape:
		return actionExit
	case tcell.KeyUp:
		s.adjustHeat(5, 1)
	case tcell.KeyDown:
		s.adjustHeat(-5, -1)
	default:
		return actionExit
	}
	return actionNone
}

func (s *screensaver) handleKeyPlayground(ev *tcell.EventKey) action {
	if ev.Key() == tcell.KeyEscape {
		return actionExit
	}
	return actionNone
}

func (s *screensaver) handleKeyLock(ev *tcell.EventKey) action {
	switch ev.Key() {
	case tcell.KeyEnter:
		if s.tryUnlock() {
			return actionExit
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		s.inputBuffer.Backspace()
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight:
		s.inputBuffer.AppendString(lock.ArrowKeyMarker(ev.Key()))
	case tcell.KeyRune:
		s.inputBuffer.AppendRune(ev.Rune())
	}
	return actionNone
}

func (s *screensaver) tryUnlock() bool {
	password := s.inputBuffer.Bytes()
	defer lock.ClearBytes(password)

	valid, err := lock.CheckPassword(password)
	if err != nil || !valid {
		s.inputBuffer.Clear()
		return false
	}
	return true
}

func (s *screensaver) adjustHeat(powerDelta, sourcesDelta int) {
	s.heatPower = clamp(s.heatPower+powerDelta, minHeat, maxHeat)
	s.heatSources = clamp(s.heatSources+sourcesDelta, minSources, s.width)
}

// ---- Rendering

func (s *screensaver) run() error {
	s.screen.Clear()
	s.screen.HideCursor()

	if s.width <= 0 || s.height <= 0 {
		return nil
	}

	go s.pollEvents()

	for {
		if done := s.processEvents(); done {
			return nil
		}
		s.updateVisualState()
		s.renderFrame()
		time.Sleep(frameDelay)
		s.frame++
	}
}

// pollEvents reads events until the screen is finalized.
// When screen.Fini() is called (in close()), PollEvent returns nil, ending this goroutine.
func (s *screensaver) pollEvents() {
	for {
		ev := s.screen.PollEvent()
		if ev == nil {
			return
		}
		s.events <- ev
	}
}

func (s *screensaver) processEvents() bool {
	for {
		select {
		case ev := <-s.events:
			if s.handleEvent(ev) == actionExit {
				return true
			}
		default:
			return false
		}
	}
}

func (s *screensaver) updateVisualState() {
	if s.visualState == nil {
		return
	}
	s.visualState.OnFrame()
	s.heatPower = s.visualState.EffectiveHeatPower()
}

func (s *screensaver) renderFrame() {
	s.generateHeat()
	s.renderFire()
	s.renderTicker()
	s.screen.Show()
}

func (s *screensaver) generateHeat() {
	bottomRow := s.width * (s.height - 1)
	for i := 0; i < s.heatSources; i++ {
		idx := rand.Intn(s.width) + bottomRow
		if idx >= 0 && idx < len(s.buffer) {
			s.buffer[idx] = s.heatPower
		}
	}
}

func (s *screensaver) renderFire() {
	size := s.width * s.height
	tickerRows := 0
	if s.haveTicker {
		tickerRows = 2
	}

	for i := 0; i < size; i++ {
		s.buffer[i] = (s.buffer[i] + s.buffer[i+1] + s.buffer[i+s.width] + s.buffer[i+s.width+1]) / 4

		row, col := i/s.width, i%s.width
		if row >= s.height || col >= s.width || row >= s.height-tickerRows {
			continue
		}

		v := s.buffer[i]
		style := s.styleForValue(v)
		char := s.theme.chars[clamp(v, 0, 9)]
		s.screen.SetContent(col, row, char, nil, style)
	}
}

func (s *screensaver) styleForValue(v int) tcell.Style {
	switch {
	case v > 15:
		return s.theme.styles[4]
	case v > 9:
		return s.theme.styles[3]
	case v > 4:
		return s.theme.styles[2]
	default:
		return s.theme.styles[1]
	}
}

func (s *screensaver) renderTicker() {
	if !s.haveTicker || s.height < 2 || len(s.msgText) == 0 {
		return
	}

	msgRunes := []rune(s.msgText)
	metaRunes := []rune(s.metaText)
	msgRow := s.height - 2
	metaRow := s.height - 1
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite)

	for x := 0; x < s.width; x++ {
		mi := (s.tickerOffset + x) % len(msgRunes)
		mj := (s.tickerOffset + x) % len(metaRunes)
		s.screen.SetContent(x, msgRow, msgRunes[mi], nil, style)
		s.screen.SetContent(x, metaRow, metaRunes[mj], nil, style)
	}

	if s.frame%4 == 0 {
		s.tickerOffset = (s.tickerOffset + 1) % len(msgRunes)
	}
}

// ---- Command Execution

func execScreensaver(cfg screensaverConfig) error {
	if cfg.mode == ModeLock && !lock.PasswordExists() {
		return fmt.Errorf("no password configured. Run 'yule-log lock set-password' first")
	}

	s, err := newScreensaver(cfg)
	if err != nil {
		return err
	}
	defer s.close()

	return s.run()
}

type idleConfig struct {
	Timeout  int
	Once     bool
	Contribs bool
	NoTicker bool
}

func execIdle(cfg idleConfig) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}

	if cfg.Once {
		triggerScreensaver(context.Background(), exePath, cfg.Contribs, cfg.NoTicker)
		return nil
	}

	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("not running inside tmux")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("Yule log idle watcher started (timeout: %ds, poll: %ds)\n", cfg.Timeout, pollInterval)

	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()

	waitingForActivity := false

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Yule log idle watcher stopped")
			return nil
		case <-ticker.C:
			idleSeconds, err := getClientIdleTime(ctx)
			if err != nil {
				continue
			}

			if waitingForActivity {
				if idleSeconds < cfg.Timeout {
					waitingForActivity = false
				}
				continue
			}

			if idleSeconds >= cfg.Timeout {
				triggerScreensaver(ctx, exePath, cfg.Contribs, cfg.NoTicker)
				waitingForActivity = true
			}
		}
	}
}

type lockConfig struct {
	SocketProtect bool
	Contribs      bool
	NoTicker      bool
	Cooldown      fire.CooldownSpeed
}

func execLock(cfg lockConfig) error {
	if !lock.PasswordExists() {
		return fmt.Errorf("no password configured. Run 'yule-log lock set-password' first")
	}

	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("not running inside tmux")
	}

	var socketPath string
	var originalPerm os.FileMode

	if cfg.SocketProtect {
		var err error
		socketPath, err = lock.GetTmuxSocketPath()
		if err != nil {
			return fmt.Errorf("getting tmux socket: %w", err)
		}

		originalPerm, err = lock.RestrictSocket(socketPath)
		if err != nil {
			return fmt.Errorf("restricting socket: %w", err)
		}
		defer lock.RestoreSocket(socketPath, originalPerm)
	}

	if err := lock.Lock(socketPath, originalPerm); err != nil {
		return fmt.Errorf("creating lock state: %w", err)
	}
	defer lock.Unlock()

	return execScreensaver(screensaverConfig{
		mode:     ModeLock,
		contribs: cfg.Contribs,
		noTicker: cfg.NoTicker,
		cooldown: cfg.Cooldown,
	})
}

func execSetPassword() error {
	reader := bufio.NewReader(os.Stdin)

	if lock.PasswordExists() {
		fmt.Print("A password is already set. Replace it? [y/N]: ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Password not changed.")
			return nil
		}
	}

	fmt.Println("Set your lock password.")
	fmt.Println("You can use regular characters and arrow keys (shown as arrows).")
	fmt.Print("Enter password: ")

	password, err := readPasswordWithArrows()
	if err != nil {
		return fmt.Errorf("reading password: %w", err)
	}
	if len(password) == 0 {
		return fmt.Errorf("password cannot be empty")
	}
	defer lock.ClearBytes(password)

	fmt.Print("\nConfirm password: ")
	confirm, err := readPasswordWithArrows()
	if err != nil {
		return fmt.Errorf("reading confirmation: %w", err)
	}
	defer lock.ClearBytes(confirm)

	if string(password) != string(confirm) {
		return fmt.Errorf("passwords do not match")
	}

	if err := lock.SavePassword(password); err != nil {
		return fmt.Errorf("saving password: %w", err)
	}

	fmt.Println("\nPassword set successfully.")
	return nil
}

func execLockStatus() error {
	if lock.PasswordExists() {
		fmt.Println("Password: configured")
	} else {
		fmt.Println("Password: not configured")
	}

	if lock.IsLocked() {
		if duration, err := lock.LockDuration(); err == nil {
			fmt.Printf("Status: locked (for %s)\n", duration.Round(time.Second))
		} else {
			fmt.Println("Status: locked")
		}
	} else {
		fmt.Println("Status: unlocked")
	}

	return nil
}

// ---- Helpers

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func getClientIdleTime(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "tmux", "display-message", "-p", "#{client_activity}")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("get client activity: %w", err)
	}

	activityStr := strings.TrimSpace(string(out))
	if activityStr == "" {
		return 0, fmt.Errorf("empty activity timestamp")
	}

	activityTime, err := strconv.ParseInt(activityStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse activity timestamp: %w", err)
	}

	return max(int(time.Now().Unix()-activityTime), 0), nil
}

func triggerScreensaver(ctx context.Context, exePath string, contribs, noTicker bool) {
	args := []string{exePath, "run"}
	if contribs {
		args = append(args, "--contribs")
	}
	if noTicker {
		args = append(args, "--no-ticker")
	}

	panePathCmd := exec.CommandContext(ctx, "tmux", "display-message", "-p", "#{pane_current_path}")
	if panePathOut, _ := panePathCmd.Output(); len(panePathOut) > 0 {
		if panePath := strings.TrimSpace(string(panePathOut)); panePath != "" {
			args = append(args, "--dir", panePath)
		}
	}

	cmd := exec.Command("tmux", "display-popup", "-E", "-w", "100%", "-h", "100%", strings.Join(args, " "))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	_ = cmd.Run() // Ignore error: popup may fail if tmux is unavailable
}

// ---- Git Ticker

func buildGitTickerText(maxCommits int, gitDir string) (string, string, bool) {
	cmd := exec.Command("git", "log", "-n", strconv.Itoa(maxCommits), "--pretty=format:%h%x09%an%x09%ar%x09%s")

	if gitDir != "" {
		cmd.Dir = gitDir
	} else if dir := os.Getenv("YULE_LOG_GIT_DIR"); dir != "" {
		cmd.Dir = dir
	}

	out, err := cmd.Output()
	if err != nil {
		return "", "", false
	}
	return parseGitLogToTicker(string(out))
}

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

		author, relTime, subject := parts[1], parts[2], parts[3]
		meta := "by " + author + " " + relTime

		width := max(len([]rune(subject)), len([]rune(meta))) + 4
		msgSegs = append(msgSegs, padRight(subject, width))
		metaSegs = append(metaSegs, padRight(meta, width))
	}

	if len(msgSegs) == 0 {
		return "", "", false
	}
	return strings.Join(msgSegs, ""), strings.Join(metaSegs, ""), true
}

func padRight(s string, n int) string {
	rs := []rune(s)
	if len(rs) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(rs))
}

// ---- Password Input

func readPasswordWithArrows() ([]byte, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	defer s.Fini()

	width, height := s.Size()
	row := height / 2
	startCol := 2

	s.Clear()
	s.SetContent(0, row, '>', nil, tcell.StyleDefault)
	s.SetContent(1, row, ' ', nil, tcell.StyleDefault)
	s.Show()

	var password []byte
	col := startCol

	for {
		ev := s.PollEvent()
		keyEv, ok := ev.(*tcell.EventKey)
		if !ok {
			continue
		}

		switch keyEv.Key() {
		case tcell.KeyEnter:
			return password, nil

		case tcell.KeyEscape:
			lock.ClearBytes(password)
			return nil, nil

		case tcell.KeyCtrlC:
			lock.ClearBytes(password)
			return nil, fmt.Errorf("interrupted")

		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(password) == 0 {
				continue
			}
			password, col = handlePasswordBackspace(s, password, col, row)

		case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight:
			password = append(password, lock.ArrowKeyMarker(keyEv.Key())...)
			arrowRune := lock.ArrowKeyDisplay(keyEv.Key())
			s.SetContent(col, row, arrowRune, nil, tcell.StyleDefault.Foreground(tcell.ColorYellow))
			col++

		case tcell.KeyRune:
			password = append(password, byte(keyEv.Rune()))
			s.SetContent(col, row, '*', nil, tcell.StyleDefault)
			col++
		}

		col = clamp(col, startCol, width-1)
		s.Show()
	}
}

func handlePasswordBackspace(s tcell.Screen, password []byte, col, row int) ([]byte, int) {
	// Handle multi-byte arrow markers
	if len(password) >= 2 {
		last2 := string(password[len(password)-2:])
		if last2 == lock.ArrowUpMarker || last2 == lock.ArrowDownMarker ||
			last2 == lock.ArrowLeftMarker || last2 == lock.ArrowRightMarker {
			password = password[:len(password)-2]
		} else {
			password = password[:len(password)-1]
		}
	} else {
		password = password[:len(password)-1]
	}

	col--
	s.SetContent(col, row, ' ', nil, tcell.StyleDefault)
	return password, col
}

// ---- CLI Setup

func main() {
	if err := run(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	rootCmd := buildCLI()
	return rootCmd.ParseAndRun(context.Background(), os.Args[1:])
}

func buildCLI() *ffcli.Command {
	// Run command
	runFlagSet := flag.NewFlagSet("yule-log run", flag.ExitOnError)
	runContribs := runFlagSet.Bool("contribs", false, "Use GitHub contribution graph-style visualization")
	runGitDir := runFlagSet.String("dir", "", "Git directory for commit ticker (defaults to current dir or YULE_LOG_GIT_DIR)")
	runNoTicker := runFlagSet.Bool("no-ticker", false, "Disable git commit ticker (fire animation only)")
	runPlayground := runFlagSet.Bool("playground", false, "Playground mode: only ESC exits, all keys affect fire")
	runCooldown := runFlagSet.String("cooldown", string(fire.DefaultCooldown), "Fire cooldown speed: fast, medium, slow")
	runLock := runFlagSet.Bool("lock", false, "Lock mode: require password to exit")

	runCmd := &ffcli.Command{
		Name:       "run",
		ShortUsage: "yule-log run [flags]",
		ShortHelp:  "Run the screensaver",
		FlagSet:    runFlagSet,
		Exec: func(_ context.Context, _ []string) error {
			mode := ModeNormal
			if *runLock {
				mode = ModeLock
			} else if *runPlayground {
				mode = ModePlayground
			}
			return execScreensaver(screensaverConfig{
				mode:     mode,
				contribs: *runContribs,
				gitDir:   *runGitDir,
				noTicker: *runNoTicker,
				cooldown: fire.CooldownSpeed(*runCooldown),
			})
		},
	}

	// Idle command
	idleFlagSet := flag.NewFlagSet("yule-log idle", flag.ExitOnError)
	idleTimeout := idleFlagSet.Int("timeout", defaultIdleTimeout, "Idle timeout in seconds before triggering screensaver")
	idleOnce := idleFlagSet.Bool("once", false, "Trigger screensaver immediately and exit")
	idleContribs := idleFlagSet.Bool("contribs", false, "Use GitHub contribution graph-style visualization")
	idleNoTicker := idleFlagSet.Bool("no-ticker", false, "Disable git commit ticker")

	idleCmd := &ffcli.Command{
		Name:       "idle",
		ShortUsage: "yule-log idle [flags]",
		ShortHelp:  "Run idle watcher daemon",
		FlagSet:    idleFlagSet,
		Exec: func(_ context.Context, _ []string) error {
			return execIdle(idleConfig{
				Timeout:  *idleTimeout,
				Once:     *idleOnce,
				Contribs: *idleContribs,
				NoTicker: *idleNoTicker,
			})
		},
	}

	// Lock command and subcommands
	lockFlagSet := flag.NewFlagSet("yule-log lock", flag.ExitOnError)
	lockSocketProtect := lockFlagSet.Bool("socket-protect", true, "Restrict tmux socket permissions during lock")
	lockContribs := lockFlagSet.Bool("contribs", false, "Use GitHub contribution graph-style visualization")
	lockNoTicker := lockFlagSet.Bool("no-ticker", false, "Disable git commit ticker")
	lockCooldown := lockFlagSet.String("cooldown", string(fire.DefaultCooldown), "Fire cooldown speed: fast, medium, slow")

	setPasswordCmd := &ffcli.Command{
		Name:       "set-password",
		ShortUsage: "yule-log lock set-password",
		ShortHelp:  "Set or update the lock password",
		Exec:       func(_ context.Context, _ []string) error { return execSetPassword() },
	}

	lockStatusCmd := &ffcli.Command{
		Name:       "status",
		ShortUsage: "yule-log lock status",
		ShortHelp:  "Show lock status",
		Exec:       func(_ context.Context, _ []string) error { return execLockStatus() },
	}

	lockCmd := &ffcli.Command{
		Name:        "lock",
		ShortUsage:  "yule-log lock [flags]",
		ShortHelp:   "Lock the tmux session",
		FlagSet:     lockFlagSet,
		Subcommands: []*ffcli.Command{setPasswordCmd, lockStatusCmd},
		Exec: func(_ context.Context, _ []string) error {
			return execLock(lockConfig{
				SocketProtect: *lockSocketProtect,
				Contribs:      *lockContribs,
				NoTicker:      *lockNoTicker,
				Cooldown:      fire.CooldownSpeed(*lockCooldown),
			})
		},
	}

	// Root command
	return &ffcli.Command{
		ShortUsage:  "yule-log [flags] <subcommand>",
		ShortHelp:   "A tmux screensaver with fire animation and git commit ticker",
		LongHelp:    "Controls:\n  Arrow Up/Down   Adjust flame intensity\n  Any other key   Exit screensaver\n\nLock mode:\n  All keys feed the fire, Enter submits password",
		FlagSet:     flag.NewFlagSet("yule-log", flag.ExitOnError),
		Subcommands: []*ffcli.Command{runCmd, idleCmd, lockCmd},
		Exec:        func(_ context.Context, _ []string) error { return execScreensaver(screensaverConfig{}) },
	}
}
