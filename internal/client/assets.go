package client

import (
	"bytes"
	"image"
	"image/color"
	"io"
	_ "image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

// Icons holds all loaded icon images
var Icons = make(map[string]*ebiten.Image)

// Audio
var audioContext *audio.Context
var winnerMusic *audio.Player
var winnerMusicBytes []byte

// LoadIcons loads all icon images from the assets/icons directory
func LoadIcons() {
	iconNames := []string{
		"stockpile", "horse", "weapon", "city", "boat",
		"coal", "gold", "iron", "timber", "grassland",
	}

	// Try to find icons directory - check common locations
	iconDirs := []string{
		"internal/client/assets/icons",
		"assets/icons",
		"data/icons",
	}

	var baseDir string
	for _, dir := range iconDirs {
		if _, err := os.Stat(dir); err == nil {
			baseDir = dir
			break
		}
	}

	if baseDir == "" {
		log.Printf("No icons directory found, will use fallback icons")
		return
	}

	for _, name := range iconNames {
		path := filepath.Join(baseDir, name+".png")
		data, err := os.ReadFile(path)
		if err != nil {
			// Not an error - icon just doesn't exist
			continue
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			log.Printf("Failed to decode icon %s: %v", name, err)
			continue
		}

		Icons[name] = ebiten.NewImageFromImage(img)
		log.Printf("Loaded icon: %s", name)
	}
}

// GetIcon returns an icon image, or nil if not loaded
func GetIcon(name string) *ebiten.Image {
	return Icons[name]
}

// CreatePlaceholderIcons creates simple colored placeholder icons
// Call this if PNG icons aren't available
func CreatePlaceholderIcons() {
	size := 16

	// Stockpile - brown box
	Icons["stockpile"] = createColoredSquare(size, color.RGBA{139, 90, 43, 255})

	// Horse - tan
	Icons["horse"] = createColoredSquare(size, color.RGBA{210, 180, 140, 255})

	// Weapon - red
	Icons["weapon"] = createColoredSquare(size, color.RGBA{200, 50, 50, 255})

	// City - white
	Icons["city"] = createColoredSquare(size, color.RGBA{240, 240, 240, 255})

	// Boat - blue
	Icons["boat"] = createColoredSquare(size, color.RGBA{100, 150, 220, 255})

	// Resources
	Icons["coal"] = createColoredSquare(size, color.RGBA{40, 40, 40, 255})
	Icons["gold"] = createColoredSquare(size, color.RGBA{255, 215, 0, 255})
	Icons["iron"] = createColoredSquare(size, color.RGBA{160, 160, 180, 255})
	Icons["timber"] = createColoredSquare(size, color.RGBA{100, 70, 40, 255})
	Icons["grassland"] = createColoredSquare(size, color.RGBA{120, 180, 80, 255}) // Green for grassland/horses
}

func createColoredSquare(size int, c color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(c)
	return img
}

// InitAudio initializes the audio context
func InitAudio() {
	audioContext = audio.NewContext(44100)
}

// LoadAudio loads audio files
func LoadAudio() {
	// Try to find sound directory
	soundDirs := []string{
		"internal/client/assets/sound",
		"assets/sound",
		"data/sound",
	}

	var baseDir string
	for _, dir := range soundDirs {
		if _, err := os.Stat(dir); err == nil {
			baseDir = dir
			break
		}
	}

	if baseDir == "" {
		log.Printf("No sound directory found")
		return
	}

	// Load winner music
	path := filepath.Join(baseDir, "winner.mp3")
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Failed to load winner.mp3: %v", err)
		return
	}
	winnerMusicBytes = data
	log.Printf("Loaded winner music: %s", path)
}

// PlayWinnerMusic starts playing the victory music
func PlayWinnerMusic() {
	if audioContext == nil || len(winnerMusicBytes) == 0 {
		log.Printf("Cannot play winner music: audio not initialized")
		return
	}

	// Stop any existing music
	StopWinnerMusic()

	// Decode the MP3
	decoded, err := mp3.DecodeWithSampleRate(44100, bytes.NewReader(winnerMusicBytes))
	if err != nil {
		log.Printf("Failed to decode winner.mp3: %v", err)
		return
	}

	// Create an infinite loop player
	loop := audio.NewInfiniteLoop(decoded, decoded.Length())

	player, err := audioContext.NewPlayer(loop)
	if err != nil {
		log.Printf("Failed to create audio player: %v", err)
		return
	}

	winnerMusic = player
	winnerMusic.Play()
	log.Printf("Playing winner music")
}

// StopWinnerMusic stops the victory music
func StopWinnerMusic() {
	if winnerMusic != nil {
		winnerMusic.Close()
		winnerMusic = nil
	}
}

// IsWinnerMusicPlaying returns true if victory music is playing
func IsWinnerMusicPlaying() bool {
	return winnerMusic != nil && winnerMusic.IsPlaying()
}

// audioLoopReader wraps a reader to loop infinitely
type audioLoopReader struct {
	src    io.ReadSeeker
	length int64
}

func (r *audioLoopReader) Read(p []byte) (n int, err error) {
	n, err = r.src.Read(p)
	if err == io.EOF {
		r.src.Seek(0, io.SeekStart)
		return r.Read(p)
	}
	return n, err
}
