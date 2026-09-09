package auth

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"sync"

	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/slide"
	"golang.org/x/image/draw"
)

const captchaWidth, captchaHeight = 300, 180

//go:embed assets/captcha-landscape.png
var captchaLandscape []byte

// 内置素材只解码和缩放一次，离线部署无需外部图片服务；生成拼图时不会修改原图。
var loadCaptchaBackground = sync.OnceValues(func() (image.Image, error) {
	source, err := png.Decode(bytes.NewReader(captchaLandscape))
	if err != nil {
		return nil, err
	}
	background := image.NewNRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
	draw.CatmullRom.Scale(background, background.Bounds(), source, source.Bounds(), draw.Src, nil)
	return background, nil
})

type puzzleChallenge struct {
	Context string `json:"context"`
	TargetX int    `json:"targetX"`
}

// 背景来自内置素材，缺口与拼图由服务端生成；返回的坐标仅初始X与固定Y。
func generatePuzzle(custom ...image.Image) (map[string]any, int, error) {
	background, err := loadCaptchaBackground()
	if err != nil {
		return nil, 0, err
	}
	if len(custom) > 0 && custom[0] != nil {
		background = custom[0]
	}
	const size = 52
	mask, shadow, overlay := image.NewNRGBA(image.Rect(0, 0, size, size)), image.NewNRGBA(image.Rect(0, 0, size, size)), image.NewNRGBA(image.Rect(0, 0, size, size))
	inside := func(x, y int) bool {
		body := x >= 5 && x < 43 && y >= 13 && y < 49
		tab := (x-24)*(x-24)+(y-13)*(y-13) < 64
		notch := (x-5)*(x-5)+(y-31)*(y-31) < 49
		return (body || tab) && !notch
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if inside(x, y) {
				mask.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
				shadow.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 135})
				if !inside(x-1, y) || !inside(x+1, y) || !inside(x, y-1) || !inside(x, y+1) {
					overlay.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 230})
				}
			}
		}
	}
	builder := slide.NewBuilder(slide.WithImageSize(option.Size{Width: captchaWidth, Height: captchaHeight}), slide.WithRangeGraphSize(option.RangeVal{Min: size, Max: size}), slide.WithRangeDeadZoneDirections([]slide.DeadZoneDirectionType{slide.DeadZoneDirectionTypeLeft}))
	builder.SetResources(slide.WithBackgrounds([]image.Image{background}), slide.WithGraphImages([]*slide.GraphImage{{OverlayImage: overlay, ShadowImage: shadow, MaskImage: mask}}))
	data, err := builder.Make().Generate()
	if err != nil {
		return nil, 0, err
	}
	master, err := data.GetMasterImage().ToBase64()
	if err != nil {
		return nil, 0, err
	}
	thumb, err := data.GetTileImage().ToBase64()
	if err != nil {
		return nil, 0, err
	}
	block := data.GetData()
	return map[string]any{"image": master, "thumb": thumb, "width": captchaWidth, "height": captchaHeight, "thumbWidth": block.Width, "thumbHeight": block.Height, "thumbX": 0, "thumbY": block.Y}, block.X, nil
}
