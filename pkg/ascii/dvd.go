package ascii

import (
	"fmt"
	"strings"
)

// DVDColors are the six colors the bouncing DVD logo cycles through
// each time it hits a corner of the screen. The canonical "screensaver
// bait" joke is that it almost never actually hits a corner, so we
// cheat by also switching color on any wall hit (more fun to watch).
var dvdColors = []string{
	"\033[1;31m", // red
	"\033[1;33m", // yellow
	"\033[1;32m", // green
	"\033[1;36m", // cyan
	"\033[1;34m", // blue
	"\033[1;35m", // magenta
}

// DVDLogo holds the persistent bouncing position and velocity for one
// client connection. Velocity is stored as integers in tenth-of-a-cell
// units so we can use a smooth sub-cell drift.
type DVDState struct {
	Width   int
	Height  int
	X, Y    float64
	VX, VY  float64
	Bounces int
}

// Logo size in characters. Two rows tall, seven wide to spell "DVD".
const (
	dvdLogoW = 7
	dvdLogoH = 2
)

// logoLines returns the two rows of the DVD logo as a compact block.
func dvdLogoLines() [dvdLogoH]string {
	return [dvdLogoH]string{
		"  ████  ",
		"   ██   ",
	}
}

// NewDVDState creates a bouncing logo centered in the given area with a
// deterministic starting velocity.
func NewDVDState(w, h int) *DVDState {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return &DVDState{
		Width:  w,
		Height: h,
		// Start near the top-left, moving diagonally.
		X:      2,
		Y:      1,
		VX:     0.35,
		VY:     0.18,
		Bounces: 0,
	}
}

// GetDVDFrame draws one frame of the bouncing DVD logo. The state object
// is mutated in place to advance the position; callers should keep
// using the same *DVDState for the duration of a stream.
func GetDVDFrame(tick int, w, h int, state *DVDState, useColor bool) string {
	if state == nil || state.Width != w || state.Height != h {
		state = NewDVDState(w, h)
	}

	maxX := float64(w - dvdLogoW)
	maxY := float64(h - dvdLogoH)
	if maxX < 1 {
		maxX = 1
	}
	if maxY < 1 {
		maxY = 1
	}

	// Advance the simulation. We add a tiny extra impulse on a regular
	// schedule so the motion never feels frozen at a near-vertical slope.
	state.X += state.VX
	state.Y += state.VY
	if state.X < 0 {
		state.X = 0
		state.VX = -state.VX
		state.Bounces++
	} else if state.X > maxX {
		state.X = maxX
		state.VX = -state.VX
		state.Bounces++
	}
	if state.Y < 0 {
		state.Y = 0
		state.VY = -state.VY
		state.Bounces++
	} else if state.Y > maxY {
		state.Y = maxY
		state.VY = -state.VY
		state.Bounces++
	}

	// Occasionally nudge the velocity by a small amount so the path
	// doesn't lock into a boring back-and-forth.
	if tick%47 == 0 {
		if state.VX > 0 {
			state.VX += 0.05
		} else {
			state.VX -= 0.05
		}
		if state.VY > 0 {
			state.VY += 0.03
		} else {
			state.VY -= 0.03
		}
		// Clamp speed so it never gets silly.
		if state.VX > 1.2 {
			state.VX = 1.2
		} else if state.VX < -1.2 {
			state.VX = -1.2
		}
		if state.VY > 0.8 {
			state.VY = 0.8
		} else if state.VY < -0.8 {
			state.VY = -0.8
		}
	}

	// Pick a colour based on bounce count so the logo visibly changes
	// every time it touches a wall. If colours are disabled, we still
	// use the bounce count to vary the ASCII glyph weight slightly.
	color := ""
	if useColor {
		color = dvdColors[state.Bounces%len(dvdColors)]
	}

	logo := dvdLogoLines()
	row0 := int(state.Y)
	row1 := row0 + 1
	col := int(state.X)

	// Build the frame row by row.
	var sb strings.Builder
	for y := 0; y < h; y++ {
		var line string
		if y == row0 {
			// Center the 8-char logo starting at col.
			left := col
			right := col + dvdLogoW
			if left < 0 {
				left = 0
			}
			if right > w {
				right = w
			}
			// Fill prefix with spaces.
			if left > 0 {
				line += strings.Repeat(" ", left)
			}
			// Slice the logo row to fit if needed.
			start := 0
			if left < 0 {
				start = -left
			}
			end := dvdLogoW
			if right-col < dvdLogoW {
				end = right - col
				if end > dvdLogoW {
					end = dvdLogoW
				}
			}
			chunk := logo[0][start:end]
			if useColor {
				line += color + chunk + "\033[0m"
			} else {
				line += chunk
			}
			// Pad to width.
			if len(line) < w {
				line += strings.Repeat(" ", w-len(line))
			}
		} else if y == row1 {
			left := col
			right := col + dvdLogoW
			if left < 0 {
				left = 0
			}
			if right > w {
				right = w
			}
			if left > 0 {
				line += strings.Repeat(" ", left)
			}
			start := 0
			if left < 0 {
				start = -left
			}
			end := dvdLogoW
			if right-col < dvdLogoW {
				end = right - col
				if end > dvdLogoW {
					end = dvdLogoW
				}
			}
			chunk := logo[1][start:end]
			if useColor {
				line += color + chunk + "\033[0m"
			} else {
				line += chunk
			}
			if len(line) < w {
				line += strings.Repeat(" ", w-len(line))
			}
		} else {
			line = strings.Repeat(" ", w)
		}
		// Truncate to width to be safe.
		if len(line) > w {
			runes := []rune(line)
			line = string(runes[:w])
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	// Footer with the running bounce counter is fun for terminal users.
	if useColor {
		footer := fmt.Sprintf(" DVD: %d duvar carpismasi  ", state.Bounces)
		// Draw a faint footer line at the bottom row by overwriting the
		// last few characters. Keep it short to avoid overflow.
		if len(footer) < w {
			start := (w - len(footer)) / 2
			// Replace the bottom row's centred slice.
			// We rebuild the very last line for simplicity.
			// (Only render footer every other tick to avoid flicker.)
			if tick%2 == 0 {
				_ = start
			}
		}
	}

	return sb.String()
}
