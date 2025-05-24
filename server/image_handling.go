package server

import (
    "fmt"
    "golang.org/x/image/draw"
    "image"
    "image/color"
)

const MAX_WIDTH = 175
const MAX_HEIGHT = 20

type ColoredSymbol struct {
    Symbol rune
    Color  string
}

type TextImage struct {
    ColoredText [][]ColoredSymbol
}

func MakeTextImage(im image.Image) *TextImage { // Can be added options in future

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

    rows := make([][]ColoredSymbol, chars_h)

    for c_y := 0; c_y < chars_h; c_y++ {
        rows[c_y] = make([]ColoredSymbol, chars_w)
        for c_x := 0; c_x < chars_w; c_x++ {
            rgba := col.RGBAAt(c_x, c_y)
            if rgba.A == 0 {
                rgba.R = 255
                rgba.G = 255
                rgba.B = 255
            }

            rows[c_y][c_x] = ColoredSymbol{
                Symbol: '🭴', //rune(base + index),
                Color:  hexColor(color.RGBA{R: rgba.R, G: rgba.G, B: rgba.B}),
            }
        }
    }

    return &TextImage{rows}

}

func hexColor(c color.Color) string {
    rgba := color.RGBAModel.Convert(c).(color.RGBA)
    return fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)
}
