package ascii

import (
	"strings"

	frames "ascii/pkg/ascii/asciilive"
)

// liveFrameState holds per-slug counters so we can do per-tick frame
// selection even when the caller passes arbitrary tick values.
type liveFrameState struct {
	tick int
}

var liveStates = map[string]*liveFrameState{}

// getLiveFrame returns the next frame for a given ascii-live slug,
// cropping or padding the output to fit the requested width/height.
// If useColor is true, the frame's high-saturation characters
// (non-space, non-punctuation) get a soft terminal green tint.
func getLiveFrame(slug string, tick, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	ft, ok := frames.FrameMap[slug]
	if !ok {
		return ""
	}

	state, ok := liveStates[slug]
	if !ok {
		state = &liveFrameState{}
		liveStates[slug] = state
	}
	state.tick = tick

	frame := ft.GetFrame(state.tick)
	if frame == "" {
		return ""
	}

	lines := strings.Split(frame, "\n")
	out := make([]string, 0, h)
	for i := 0; i < h; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		// Strip trailing newline / spaces, then pad or crop to width.
		line = strings.TrimRight(line, " \t\r")
		if len(line) > w {
			// Trim runes safely to keep multi-byte widths reasonable.
			runes := []rune(line)
			line = string(runes[:w])
		} else if len(line) < w {
			line = line + strings.Repeat(" ", w-len(line))
		}
		out = append(out, line)
	}

	content := strings.Join(out, "\n")
	if !useColor {
		return content
	}

	// Soft tint: brighten the most "drawn" characters. We only wrap
	// non-space runes; the caller can still set ?color=false to disable.
	var sb strings.Builder
	sb.Grow(len(content) * 2)
	inEscape := false
	for _, r := range content {
		if r == 0x1b {
			inEscape = true
			sb.WriteRune(r)
			continue
		}
		if inEscape {
			sb.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if r == ' ' {
			sb.WriteRune(r)
			continue
		}
		sb.WriteString("\033[1;37m")
		sb.WriteRune(r)
		sb.WriteString("\033[0m")
	}
	return sb.String()
}

// ============================================================================
// Public wrappers — one per ascii-live FrameMap slug.
// Each forwards to getLiveFrame and keeps the signature used by
// main.go's dispatch switch.
// ============================================================================

func GetKittyFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("kitty", tick, w, h, useColor)
}

func GetParrotFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("parrot", tick, w, h, useColor)
}

func GetCoinLiveFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("coin", tick, w, h, useColor)
}

func GetForrestFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("forrest", tick, w, h, useColor)
}

func GetBombFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("bomb", tick, w, h, useColor)
}

func GetNyanLiveFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("nyan", tick, w, h, useColor)
}

func GetPurdueFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("purdue", tick, w, h, useColor)
}

func GetIndiaFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("india", tick, w, h, useColor)
}

func GetKnotFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("knot", tick, w, h, useColor)
}

func GetMaxwellFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("maxwell", tick, w, h, useColor)
}

func GetEarthLiveFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("earth", tick, w, h, useColor)
}

func GetAstrendFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("as", tick, w, h, useColor)
}

func GetBrittanyFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("brittany", tick, w, h, useColor)
}

func GetBatmanFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("batman", tick, w, h, useColor)
}

func GetBNRFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("bnr", tick, w, h, useColor)
}

func GetBatmanRunningFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("batman-running", tick, w, h, useColor)
}

func GetSpidyswingFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("spidyswing", tick, w, h, useColor)
}

func GetRickLiveFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("rick", tick, w, h, useColor)
}

func GetCanYouHearMeFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("can-you-hear-me", tick, w, h, useColor)
}

func GetHESFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("hes", tick, w, h, useColor)
}

func GetDonutLiveFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("donut", tick, w, h, useColor)
}

func GetClockLiveFrame(tick, w, h int, useColor bool) string {
	return getLiveFrame("clock", tick, w, h, useColor)
}
