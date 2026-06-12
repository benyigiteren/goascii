package ascii

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	_ "image/jpeg" // Support JPEG decoding
	"image/png"    // Support PNG decoding (used for FFmpeg frame extraction)
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

type ConvertedAnimation struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	FrameDelayMs int      `json:"frame_delay_ms"`
	Delays       []int    `json:"delays"`
	Frames       []string `json:"frames"`
}

// cropToCenter crops the image to a centered square
func cropToCenter(img image.Image) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	
	size := w
	if h < w {
		size = h
	}
	
	x0 := bounds.Min.X + (w-size)/2
	y0 := bounds.Min.Y + (h-size)/2
	
	rect := image.Rect(0, 0, size, size)
	dst := image.NewRGBA(rect)
	draw.Draw(dst, rect, img, image.Pt(x0, y0), draw.Src)
	return dst
}

// ConvertGIF converts raw GIF byte data into an ASCII ConvertedAnimation structure
func ConvertGIF(slug, name string, gifData []byte, targetWidth int, cropCenter bool) (*ConvertedAnimation, error) {
	if targetWidth <= 0 {
		targetWidth = 80
	}

	reader := bytes.NewReader(gifData)
	g, err := gif.DecodeAll(reader)
	if err != nil {
		return nil, err
	}

	if len(g.Image) == 0 {
		return nil, errors.New("GIF dosyasında hiç kare bulunamadı")
	}

	width := g.Config.Width
	height := g.Config.Height

	if width <= 0 || height <= 0 {
		width = g.Image[0].Bounds().Dx()
		height = g.Image[0].Bounds().Dy()
	}

	// Aspect ratio calculations
	var aspectRatio float64
	if cropCenter {
		aspectRatio = 1.0 // Square aspect ratio
	} else {
		aspectRatio = float64(width) / float64(height)
	}

	// Smart Terminal Scaling: Ensure target height does not exceed 24 rows
	// to prevent terminal wrapping or vertical overflow!
	maxHeight := 24
	targetHeight := int(float64(targetWidth) / (aspectRatio * 2.0))
	if targetHeight > maxHeight {
		targetHeight = maxHeight
		targetWidth = int(float64(targetHeight) * aspectRatio * 2.0)
	}
	if targetHeight <= 0 {
		targetHeight = 1
	}

	var frames []string
	var delays []int
	totalDelay := 0

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Detailed character ramp for rich shading (letters, math symbols, brackets)
	ramp := " .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZ*#MW&8%B@$"

	for i, frame := range g.Image {
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		var workingImg image.Image = canvas
		if cropCenter {
			workingImg = cropToCenter(canvas)
		}

		resized := resizeNearestNeighbor(workingImg, targetWidth, targetHeight)

		// 1. First pass: calculate brightness values and find min/max for contrast stretching
		brightnessValues := make([][]float64, targetHeight)
		minB := 255.0
		maxB := 0.0

		for y := 0; y < targetHeight; y++ {
			brightnessValues[y] = make([]float64, targetWidth)
			for x := 0; x < targetWidth; x++ {
				c := resized.At(x, y)
				r, g, b, _ := c.RGBA()
				r8 := float64(r >> 8)
				g8 := float64(g >> 8)
				b8 := float64(b >> 8)
				
				bVal := 0.299*r8 + 0.587*g8 + 0.114*b8
				brightnessValues[y][x] = bVal
				
				if bVal < minB {
					minB = bVal
				}
				if bVal > maxB {
					maxB = bVal
				}
			}
		}

		// 2. Second pass: Apply linear contrast stretching and map to ASCII characters
		var buf bytes.Buffer
		for y := 0; y < targetHeight; y++ {
			for x := 0; x < targetWidth; x++ {
				bVal := brightnessValues[y][x]
				
				// Stretch contrast (histogram equalization wrapper)
				if maxB > minB {
					bVal = (bVal - minB) / (maxB - minB) * 255.0
				}

				// Map to character
				charIdx := int(bVal * float64(len(ramp)-1) / 255.0)
				if charIdx >= len(ramp) {
					charIdx = len(ramp) - 1
				}

				buf.WriteByte(ramp[charIdx])
			}
			buf.WriteByte('\n')
		}

		frames = append(frames, buf.String())

		delayMs := g.Delay[i] * 10
		if delayMs <= 0 {
			delayMs = 100
		}
		delays = append(delays, delayMs)
		totalDelay += delayMs
	}

	avgDelay := 100
	if len(g.Image) > 0 {
		avgDelay = totalDelay / len(g.Image)
	}

	return &ConvertedAnimation{
		Slug:         slug,
		Name:         name,
		FrameDelayMs: avgDelay,
		Delays:       delays,
		Frames:       frames,
	}, nil
}

// ConvertMP4 extracts frames using ffmpeg and converts them to ASCII
func ConvertMP4(slug, name string, mp4Data []byte, targetWidth int, cropCenter bool) (*ConvertedAnimation, error) {
	if targetWidth <= 0 {
		targetWidth = 80
	}

	// 1. Verify FFmpeg is installed on system
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, errors.New("MP4 dönüştürme özelliği sunucuda FFmpeg kurulu olmasını gerektirir. Lütfen FFmpeg kurun veya GIF yükleyin.")
	}

	// 2. Create temp directory
	tempDir, err := os.MkdirTemp("", "goascii_mp4_*")
	if err != nil {
		return nil, fmt.Errorf("geçici dizin oluşturulamadı: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 3. Write MP4 to temp file
	tempMP4Path := filepath.Join(tempDir, "input.mp4")
	if err := os.WriteFile(tempMP4Path, mp4Data, 0644); err != nil {
		return nil, fmt.Errorf("mp4 dosyası yazılamadı: %v", err)
	}

	// 4. Run FFmpeg command to extract frames
	// Extract at 10 frames per second (100ms delay)
	framePattern := filepath.Join(tempDir, "frame_%04d.png")
	cmd := exec.Command("ffmpeg", "-i", tempMP4Path, "-r", "10", "-f", "image2", framePattern)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("FFmpeg dönüştürme hatası: %v (Detay: %s)", err, stderr.String())
	}

	// 5. Read extracted PNG frame files
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("geçici dizin okunamadı: %v", err)
	}

	var pngFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".png" {
			pngFiles = append(pngFiles, filepath.Join(tempDir, file.Name()))
		}
	}

	if len(pngFiles) == 0 {
		return nil, errors.New("FFmpeg ile hiçbir video karesi çıkarılamadı")
	}

	// Sort file paths alphabetically (e.g. frame_0001.png, frame_0002.png...)
	sort.Strings(pngFiles)

	// Limit to max 120 frames to avoid file bloating and out of memory issues
	if len(pngFiles) > 120 {
		pngFiles = pngFiles[:120]
	}

	// 6. Decode PNG files and convert to ASCII
	var frames []string
	var delays []int
	ramp := " .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZ*#MW&8%B@$"
	
	// Determine height based on aspect ratio of first frame
	firstFile, err := os.Open(pngFiles[0])
	if err != nil {
		return nil, err
	}
	firstImg, err := png.Decode(firstFile)
	firstFile.Close()
	if err != nil {
		return nil, fmt.Errorf("ilk video karesi çözümlenemedi: %v", err)
	}

	width := firstImg.Bounds().Dx()
	height := firstImg.Bounds().Dy()

	var aspectRatio float64
	if cropCenter {
		aspectRatio = 1.0
	} else {
		aspectRatio = float64(width) / float64(height)
	}

	// Smart Terminal Scaling: Ensure target height does not exceed 24 rows
	// to prevent terminal wrapping or vertical overflow!
	maxHeight := 24
	targetHeight := int(float64(targetWidth) / (aspectRatio * 2.0))
	if targetHeight > maxHeight {
		targetHeight = maxHeight
		targetWidth = int(float64(targetHeight) * aspectRatio * 2.0)
	}
	if targetHeight <= 0 {
		targetHeight = 1
	}

	for _, pngPath := range pngFiles {
		file, err := os.Open(pngPath)
		if err != nil {
			continue
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			continue
		}

		var workingImg image.Image = img
		if cropCenter {
			workingImg = cropToCenter(img)
		}

		resized := resizeNearestNeighbor(workingImg, targetWidth, targetHeight)

		// Calculate brightness and apply contrast stretch
		brightnessValues := make([][]float64, targetHeight)
		minB := 255.0
		maxB := 0.0

		for y := 0; y < targetHeight; y++ {
			brightnessValues[y] = make([]float64, targetWidth)
			for x := 0; x < targetWidth; x++ {
				c := resized.At(x, y)
				r, g, b, _ := c.RGBA()
				r8 := float64(r >> 8)
				g8 := float64(g >> 8)
				b8 := float64(b >> 8)
				
				bVal := 0.299*r8 + 0.587*g8 + 0.114*b8
				brightnessValues[y][x] = bVal
				
				if bVal < minB {
					minB = bVal
				}
				if bVal > maxB {
					maxB = bVal
				}
			}
		}

		var buf bytes.Buffer
		for y := 0; y < targetHeight; y++ {
			for x := 0; x < targetWidth; x++ {
				bVal := brightnessValues[y][x]
				
				if maxB > minB {
					bVal = (bVal - minB) / (maxB - minB) * 255.0
				}

				charIdx := int(bVal * float64(len(ramp)-1) / 255.0)
				if charIdx >= len(ramp) {
					charIdx = len(ramp) - 1
				}

				buf.WriteByte(ramp[charIdx])
			}
			buf.WriteByte('\n')
		}

		frames = append(frames, buf.String())
		delays = append(delays, 100) // 10 fps -> 100ms
	}

	return &ConvertedAnimation{
		Slug:         slug,
		Name:         name,
		FrameDelayMs: 100,
		Delays:       delays,
		Frames:       frames,
	}, nil
}

// SaveAnimationToFile serializes the converted animation and saves it to data/animations/slug.json
func SaveAnimationToFile(dbDir string, anim *ConvertedAnimation) error {
	animDir := filepath.Join(dbDir, "animations")
	if err := os.MkdirAll(animDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(animDir, anim.Slug+".json")
	data, err := json.Marshal(anim)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// LoadAnimationFromFile reads the converted animation from disk
func LoadAnimationFromFile(dbDir, slug string) (*ConvertedAnimation, error) {
	filePath := filepath.Join(dbDir, "animations", slug+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var anim ConvertedAnimation
	if err := json.Unmarshal(data, &anim); err != nil {
		return nil, err
	}

	return &anim, nil
}

// Helper: Resize image using nearest-neighbor scaling
func resizeNearestNeighbor(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW == 0 || srcH == 0 {
		return dst
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcX := bounds.Min.X + (x * srcW / w)
			srcY := bounds.Min.Y + (y * srcH / h)
			if srcX >= bounds.Max.X {
				srcX = bounds.Max.X - 1
			}
			if srcY >= bounds.Max.Y {
				srcY = bounds.Max.Y - 1
			}
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	return dst
}

// ConvertColorToGrayscale represents a helper function to convert standard image.Color to Grayscale value
func ConvertColorToGrayscale(c color.Color) uint8 {
	r, g, b, _ := c.RGBA()
	r8 := float64(r >> 8)
	g8 := float64(g >> 8)
	b8 := float64(b >> 8)
	brightness := 0.299*r8 + 0.587*g8 + 0.114*b8
	return uint8(brightness)
}
