package fire

import "math"

// ---- Color Shift Utilities

// ApplyRedShift shifts fire colors toward bright red based on intensity (0-1).
// Used for wrong password animation - makes fire glow angry red.
func ApplyRedShift(r, g, b uint8, intensity float64) (uint8, uint8, uint8) {
	if intensity <= 0 {
		return r, g, b
	}
	if intensity > 1 {
		intensity = 1
	}

	rf := float64(r)
	gf := float64(g)
	bf := float64(b)

	// Shift toward bright red:
	// - Boost red significantly
	// - Reduce green and blue
	// - Add brightness boost at high intensity
	rf = math.Min(255, rf+intensity*100)
	gf = gf * (1 - intensity*0.7)
	bf = bf * (1 - intensity*0.8)

	// Brightness boost at high intensity for "angry glow"
	if intensity > 0.5 {
		boost := (intensity - 0.5) * 2
		rf = math.Min(255, rf+boost*50)
		gf = math.Min(255, gf+boost*20)
	}

	return uint8(rf), uint8(gf), uint8(bf)
}

// ApplyIntensityShift shifts fire colors based on intensity ratio (0-1).
// intensity = 0: original color unchanged (orange/yellow fire)
// intensity = 0.5: subtle warm-up, staying in orange range
// intensity = 0.85: shift toward red/magenta
// intensity = 1.0: shift toward blue/white (hottest)
func ApplyIntensityShift(r, g, b uint8, intensity float64) (uint8, uint8, uint8) {
	if intensity <= 0 {
		return r, g, b
	}
	if intensity > 1 {
		intensity = 1
	}

	rf := float64(r)
	gf := float64(g)
	bf := float64(b)

	// Gradual color shift based on intensity:
	// 0.0-0.5: Orange stays mostly orange, subtle saturation boost
	// 0.5-0.85: Gradual shift toward red/magenta
	// 0.85-1.0: Shift toward blue/white (only at highest intensity)

	if intensity < 0.5 {
		// Very subtle warm-up: slight saturation boost
		t := intensity / 0.5
		rf = math.Min(255, rf*(1+t*0.1))
		gf = gf * (1 - t*0.05)
	} else if intensity < 0.85 {
		// Gradual shift toward red/magenta
		t := (intensity - 0.5) / 0.35
		// Start with the 0.5 adjustments as baseline
		rf = math.Min(255, rf*1.1)
		gf = gf * 0.95
		// Then add red/magenta shift
		gf = gf * (1 - t*0.6)       // Reduce green more
		bf = math.Min(255, bf+t*60) // Add some blue for magenta tint
	} else {
		// Shift toward blue/white (only at very high intensity)
		t := (intensity - 0.85) / 0.15
		// Start from the 0.85 baseline
		rf = math.Min(255, rf*1.1)
		gf = gf * 0.95 * 0.4        // Green already reduced
		bf = math.Min(255, bf+60)   // Blue from magenta stage
		// Now shift to blue/white
		rf = rf * (1 - t*0.4)         // Reduce red
		gf = math.Min(255, gf+t*80)   // Add green for white/cyan
		bf = math.Min(255, bf+t*120)  // Boost blue significantly
	}

	// Boost overall brightness at very high intensity for "white hot" look
	if intensity > 0.9 {
		brightBoost := (intensity - 0.9) / 0.1
		rf = math.Min(255, rf+brightBoost*50)
		gf = math.Min(255, gf+brightBoost*50)
		bf = math.Min(255, bf+brightBoost*50)
	}

	return uint8(rf), uint8(gf), uint8(bf)
}
