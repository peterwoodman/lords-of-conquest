package client

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	_ "image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

// Icons holds all loaded icon images
var Icons = make(map[string]*ebiten.Image)

// Title screen images
var titleScreen8Bit *ebiten.Image
var titleScreenModern *ebiten.Image

// Audio
var audioContext *audio.Context
var winnerMusic *audio.Player
var winnerMusicBytes []byte
var introMusic *audio.Player
var introMusicBytes []byte
var bridgeSounds [][]byte // bridge1.ogg to bridge5.ogg

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

// LoadTitleScreens loads the title screen images
func LoadTitleScreens() {
	// Try to find assets directory
	assetDirs := []string{
		"internal/client/assets",
		"assets",
		"data",
	}

	var baseDir string
	for _, dir := range assetDirs {
		if _, err := os.Stat(dir); err == nil {
			baseDir = dir
			break
		}
	}

	if baseDir == "" {
		log.Printf("No assets directory found for title screens")
		return
	}

	// Load 8-bit title screen (GIF)
	path8Bit := filepath.Join(baseDir, "8-bit-title-screen.gif")
	data8Bit, err := os.ReadFile(path8Bit)
	if err != nil {
		log.Printf("Failed to load 8-bit title screen: %v", err)
	} else {
		// Decode GIF - use first frame
		gifImg, err := gif.DecodeAll(bytes.NewReader(data8Bit))
		if err != nil {
			log.Printf("Failed to decode 8-bit title screen GIF: %v", err)
		} else if len(gifImg.Image) > 0 {
			titleScreen8Bit = ebiten.NewImageFromImage(gifImg.Image[0])
			log.Printf("Loaded 8-bit title screen: %s", path8Bit)
		}
	}

	// Load modern title screen (PNG)
	pathModern := filepath.Join(baseDir, "title-screen.png")
	dataModern, err := os.ReadFile(pathModern)
	if err != nil {
		log.Printf("Failed to load modern title screen: %v", err)
	} else {
		img, _, err := image.Decode(bytes.NewReader(dataModern))
		if err != nil {
			log.Printf("Failed to decode modern title screen: %v", err)
		} else {
			titleScreenModern = ebiten.NewImageFromImage(img)
			log.Printf("Loaded modern title screen: %s", pathModern)
		}
	}
}

// GetTitleScreen8Bit returns the 8-bit title screen image
func GetTitleScreen8Bit() *ebiten.Image {
	return titleScreen8Bit
}

// GetTitleScreenModern returns the modern title screen image
func GetTitleScreenModern() *ebiten.Image {
	return titleScreenModern
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
	} else {
		winnerMusicBytes = data
		log.Printf("Loaded winner music: %s", path)
	}

	// Load intro music
	introPath := filepath.Join(baseDir, "intro.ogg")
	introData, err := os.ReadFile(introPath)
	if err != nil {
		log.Printf("Failed to load intro.ogg: %v", err)
	} else {
		introMusicBytes = introData
		log.Printf("Loaded intro music: %s", introPath)
	}

	// Load bridge sounds (bridge1.ogg to bridge5.ogg)
	bridgeSounds = make([][]byte, 0, 5)
	for i := 1; i <= 5; i++ {
		bridgePath := filepath.Join(baseDir, fmt.Sprintf("bridge%d.ogg", i))
		bridgeData, err := os.ReadFile(bridgePath)
		if err != nil {
			log.Printf("Failed to load bridge%d.ogg: %v", i, err)
		} else {
			bridgeSounds = append(bridgeSounds, bridgeData)
			log.Printf("Loaded bridge sound: %s", bridgePath)
		}
	}
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

// PlayIntroMusic starts playing the intro music
func PlayIntroMusic() {
	if audioContext == nil || len(introMusicBytes) == 0 {
		log.Printf("Cannot play intro music: audio not initialized or file not loaded")
		return
	}

	// Stop any existing intro music
	StopIntroMusic()

	// Decode the OGG
	decoded, err := vorbis.DecodeWithSampleRate(44100, bytes.NewReader(introMusicBytes))
	if err != nil {
		log.Printf("Failed to decode intro.ogg: %v", err)
		return
	}

	// Create an infinite loop player
	loop := audio.NewInfiniteLoop(decoded, decoded.Length())

	player, err := audioContext.NewPlayer(loop)
	if err != nil {
		log.Printf("Failed to create audio player for intro: %v", err)
		return
	}

	introMusic = player
	introMusic.Play()
	log.Printf("Playing intro music")
}

// StopIntroMusic stops the intro music immediately
func StopIntroMusic() {
	if introMusic != nil {
		introMusic.Close()
		introMusic = nil
		log.Printf("Stopped intro music")
	}
}

// FadeOutIntroMusic gradually fades out the intro music over the specified duration
func FadeOutIntroMusic(durationMs int) {
	if introMusic == nil || !introMusic.IsPlaying() {
		return
	}

	log.Printf("Fading out intro music over %dms", durationMs)

	// Fade out in a goroutine
	go func() {
		player := introMusic // Capture reference
		if player == nil {
			return
		}

		steps := 30 // Number of volume steps
		stepDuration := time.Duration(durationMs/steps) * time.Millisecond
		volumeStep := 1.0 / float64(steps)

		for i := 0; i < steps; i++ {
			if player != introMusic {
				// Player changed (e.g., music restarted), abort fade
				return
			}
			volume := 1.0 - (float64(i+1) * volumeStep)
			if volume < 0 {
				volume = 0
			}
			player.SetVolume(volume)
			time.Sleep(stepDuration)
		}

		// Stop the music after fade completes
		if player == introMusic {
			StopIntroMusic()
		}
	}()
}

// IsIntroMusicPlaying returns true if intro music is playing
func IsIntroMusicPlaying() bool {
	return introMusic != nil && introMusic.IsPlaying()
}

// PlayBridgeSound plays a random bridge sound
func PlayBridgeSound() {
	if audioContext == nil || len(bridgeSounds) == 0 {
		log.Printf("Cannot play bridge sound: audio not initialized or sounds not loaded")
		return
	}

	// Pick a random bridge sound
	idx := rand.Intn(len(bridgeSounds))
	soundData := bridgeSounds[idx]

	// Decode and play in a goroutine to not block
	go func() {
		decoded, err := vorbis.DecodeWithSampleRate(44100, bytes.NewReader(soundData))
		if err != nil {
			log.Printf("Failed to decode bridge sound: %v", err)
			return
		}

		player, err := audioContext.NewPlayer(decoded)
		if err != nil {
			log.Printf("Failed to create bridge sound player: %v", err)
			return
		}

		player.Play()
		log.Printf("Playing bridge sound %d", idx+1)
	}()
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
