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

const MAX_WIDTH = 175
const MAX_HEIGHT = 20

// a simple ramp from darkest (“@”) to lightest (“ ”)
const ASCII_BLOCK_SYMBOL = '▓'

const BLANK = '⠀'

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

    /*
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
    			if rgba.A == 0 {
    				row[x] = server.ColoredSymbol{
    					Symbol: BLANK,
    					Color:  hexColor(color.RGBA{R: rgba.R, G: rgba.G, B: rgba.B}),
    				}
    			} else {
    				row[x] = server.ColoredSymbol{
    					Symbol: ASCII_BLOCK_SYMBOL,
    					Color:  hexColor(color.RGBA{R: rgba.R, G: rgba.G, B: rgba.B}),
    				}
    			}
    		}
    		rows[y] = row
    	}
    */

    factor := min(
        float32(MAX_WIDTH*1)/float32(im.Bounds().Dx()),
        float32(MAX_HEIGHT*16)/float32(im.Bounds().Dy()),
    )

    target_w := int(float32(im.Bounds().Dx()) * factor)
    target_h := int(float32(im.Bounds().Dy()) * factor)

    chars_w := target_w / 1
    chars_h := target_h / 9

    fmt.Println(target_w, target_h, chars_w, chars_h)

    col := image.NewRGBA(image.Rect(0, 0, chars_w, chars_h))
    draw.CatmullRom.Scale(col, col.Rect, im, im.Bounds(), draw.Over, nil)

    rows := make([][]server.ColoredSymbol, chars_h)

    for c_y := 0; c_y < chars_h; c_y++ {
        rows[c_y] = make([]server.ColoredSymbol, chars_w)
        for c_x := 0; c_x < chars_w; c_x++ {
            rgba := col.RGBAAt(c_x, c_y)
            if rgba.A == 0 {
                rgba.R = 255
                rgba.G = 255
                rgba.B = 255
            }

            rows[c_y][c_x] = server.ColoredSymbol{
                Symbol: '🭴', //rune(base + index),
                Color:  hexColor(color.RGBA{R: rgba.R, G: rgba.G, B: rgba.B}),
            }
        }
    }

    return rows

}

func hexColor(c color.Color) string {
    rgba := color.RGBAModel.Convert(c).(color.RGBA)
    return fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)
}
