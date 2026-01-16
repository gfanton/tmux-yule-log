package fire

// ---- Visual Feedback Parameters

const (
	// BaseHeatPower is the resting fire intensity (lower than normal screensaver).
	BaseHeatPower = 50

	// BurstHeat is the heat added per keypress.
	BurstHeat = 15

	// MaxBurstHeat is the maximum burst accumulation.
	MaxBurstHeat = 40

	// DefaultCooldownRate is the heat decay per frame.
	DefaultCooldownRate = 2

	// DefaultCooldownDelay is frames before cooldown starts.
	DefaultCooldownDelay = 5
)

// ---- Cooldown Presets

// CooldownSpeed represents a named cooldown speed preset.
type CooldownSpeed string

const (
	CooldownFast    CooldownSpeed = "fast"
	CooldownMedium  CooldownSpeed = "medium"
	CooldownSlow    CooldownSpeed = "slow"
	DefaultCooldown CooldownSpeed = CooldownMedium
)

// CooldownPreset defines cooldown timing parameters.
type CooldownPreset struct {
	Rate  int // Heat decay per frame
	Delay int // Frames before cooldown starts
}

// CooldownPresets maps preset names to their parameters.
var CooldownPresets = map[CooldownSpeed]CooldownPreset{
	CooldownFast:   {Rate: 4, Delay: 3}, // ~0.5-1 sec cooldown
	CooldownMedium: {Rate: 2, Delay: 5}, // ~1-1.5 sec cooldown
	CooldownSlow:   {Rate: 1, Delay: 8}, // ~2-3 sec cooldown
}

// VisualState tracks the visual feedback state for lock mode.
type VisualState struct {
	// CurrentBurst is the accumulated burst heat (0 to MaxBurstHeat).
	CurrentBurst int

	// FramesSinceInput counts frames since last keypress.
	FramesSinceInput int

	// CooldownRate is the heat decay per frame.
	CooldownRate int

	// CooldownDelay is frames before cooldown starts.
	CooldownDelay int
}

// NewVisualState creates a new visual state with default parameters.
func NewVisualState() *VisualState {
	return &VisualState{
		CooldownRate:  DefaultCooldownRate,
		CooldownDelay: DefaultCooldownDelay,
	}
}

// NewVisualStateWithPreset creates a new visual state with a cooldown preset.
func NewVisualStateWithPreset(preset CooldownSpeed) *VisualState {
	vs := NewVisualState()
	if p, ok := CooldownPresets[preset]; ok {
		vs.CooldownRate = p.Rate
		vs.CooldownDelay = p.Delay
	}
	return vs
}

// OnKeyPress should be called when any key is pressed.
// It increases fire intensity and resets cooldown timer.
func (vs *VisualState) OnKeyPress() {
	vs.CurrentBurst += BurstHeat
	if vs.CurrentBurst > MaxBurstHeat {
		vs.CurrentBurst = MaxBurstHeat
	}
	vs.FramesSinceInput = 0
}

// OnFrame should be called each frame to update cooldown.
func (vs *VisualState) OnFrame() {
	vs.FramesSinceInput++

	if vs.FramesSinceInput > vs.CooldownDelay {
		vs.CurrentBurst -= vs.CooldownRate
		if vs.CurrentBurst < 0 {
			vs.CurrentBurst = 0
		}
	}
}

// EffectiveHeatPower returns the current heat power for rendering.
func (vs *VisualState) EffectiveHeatPower() int {
	return BaseHeatPower + vs.CurrentBurst
}

// Reset clears the visual state to initial values.
func (vs *VisualState) Reset() {
	vs.CurrentBurst = 0
	vs.FramesSinceInput = 0
}
