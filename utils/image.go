package utils

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gregthemadmonk/mctg-server-bot/server"
	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
)

const MAX_WIDTH = 30
const MAX_HEIGHT = 100

// a simple ramp from darkest (“@”) to lightest (“ ”)
var asciiRamp = []rune("@%#*+=-:. ")

const BLANK = ' '

func ConvertImageToColoredText(content []byte, extension string) [][]server.ColoredSymbol {
	contentBuffer := bytes.NewBuffer(content)

	var im image.Image
	var err error

	switch extension {
	case ".webp":
		im, err = webp.Decode(contentBuffer)
		break
	case ".jpg":
		im, err = jpeg.Decode(contentBuffer)
		break
	case ".png":
		im, err = png.Decode(contentBuffer)
		break
	case ".gif":
		im, err = gif.Decode(contentBuffer)
		break
	default:
		err = errors.New("unsupported image format")
		break
	}

	if err != nil {
		log.Println("Failed to decode image:", err)
		return nil
	}

	targetW := min(im.Bounds().Dx(), MAX_WIDTH)
	// Scale height same as width
	targetH := min(im.Bounds().Dy()/(im.Bounds().Dx()/targetW), MAX_HEIGHT)

	resized := image.NewRGBA(image.Rect(0, 0, targetH, targetW))

	draw.CatmullRom.Scale(resized, resized.Rect, im, im.Bounds(), draw.Over, nil)

	// This fragment is vibecode
	rows := make([][]server.ColoredSymbol, targetH)
	for y := 0; y < targetH; y++ {
		row := make([]server.ColoredSymbol, targetW)
		for x := 0; x < targetW; x++ {
			// grab pixel
			rgba := resized.RGBAAt(x, y)

			// handle zero alpha
			if rgba.A <= 10 {
				row[x] = server.ColoredSymbol{
					Symbol: BLANK,
					Color:  hexColor(color.RGBA{R: rgba.R, G: rgba.G, B: rgba.B}),
				}
			} else {
				// compute luminance (simple avg)
				lum := uint16(rgba.R) + uint16(rgba.G) + uint16(rgba.B)
				lum /= 3
				// map 0–255 to 0–len(asciiRamp)-1
				idx := int(lum) * (len(asciiRamp) - 1) / 255
				sym := asciiRamp[len(asciiRamp)-1-idx]
				row[x] = server.ColoredSymbol{
					Symbol: sym,
					Color:  hexColor(color.RGBA{R: rgba.R, G: rgba.G, B: rgba.B}),
				}
			}
		}
		rows[y] = row
	}

	return rows

}

func hexColor(c color.Color) string {
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	return fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)
}
