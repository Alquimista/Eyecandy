// Package eyecandy read/write SSA/ASS Subtitle Script
package eyecandy

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"golang.org/x/image/font"

	"github.com/Alquimista/eyecandy/asstime"
	"github.com/Alquimista/eyecandy/reader"
	"github.com/Alquimista/eyecandy/utils"

	// "github.com/Alquimista/eyecandy/writer"
	writer "github.com/Alquimista/eyecandy/writer2"
)

const (
	// AlignBottomLeft Bottom Left SSA numbered Alignment
	AlignBottomLeft int = 1 + iota
	// AlignBottomCenter Bottom Center SSA numbered Alignment
	AlignBottomCenter
	// AlignBottomRight Bottom Right SSA numbered Alignment
	AlignBottomRight

	// AlignMiddleLeft Middle Left SSA numbered Alignment
	AlignMiddleLeft
	// AlignMiddleCenter Middle Center SSA numbered Alignment
	AlignMiddleCenter
	// AlignMiddleRight Middle Right SSA numbered Alignment
	AlignMiddleRight

	// AlignTopLeft Top Left SSA numbered Alignment
	AlignTopLeft
	// AlignTopCenter Top Left SSA numbered Alignment
	AlignTopCenter
	// AlignTopRight Top Right SSA numbered Alignment
	AlignTopRight
)

var reStripTags = regexp.MustCompile(`({[\s\w\d\\-]+})*`)
var reStripTags2 = regexp.MustCompile(`({[^k]+})*`)
var reKara = regexp.MustCompile(
	`{\\k[of]?(?P<duration>\d+)` + // k duration in centiseconds
		`(?:\-)*(?P<inline>[\w\d]+)*` + // inline
		`}(?P<text>[^\{\}]*)`) //text

func StripSSATags(text string) string {
	return strings.TrimSpace(reStripTags.ReplaceAllString(text, ""))
}

func StripSSATagsNotKDur(text string) string {
	return strings.TrimSpace(reStripTags2.ReplaceAllString(text, ""))
}

func GetSyls(text string) [][]string {
	return reKara.FindAllStringSubmatch(StripSSATagsNotKDur(text), -1)
}

type Dialog struct {
	Layer     int
	StartTime int
	EndTime   int
	StyleName string
	Actor     string
	Effect    string
	Text      string
	Tags      string
	Duration  int
	MidTime   int
	Style     *reader.Style
	Comment   bool
	Width     float64
	Height    float64
	Size      [2]float64
	XFix      float64
	X         float64
	Y         float64
	Top       float64
	Middle    float64
	Bottom    float64
	Left      float64
	Center    float64
	Right     float64
}

// Line Represent the subtitle"s lines.
type Line struct {
	Dialog
	Kara       string
	SylN       int
	CharN      int
	syls       [][]string
	fontFace   font.Face
	resolution [2]int
}

// Syl Represent the subtitle"s lines.
type Syl struct {
	Dialog
	Inline string
}

// Char Represent the subtitle"s lines.
type Char struct {
	Dialog
	Inline        string
	SylStartTime  int
	SylEndTime    int
	SylMidEndTime int
	SylDuration   int
}

// Chars list all characters in a Line
func (d *Line) Chars() (chars []*Char) {

	start, end, x, dur := 0, 0, 0.0, 0
	for _, s := range d.Syls() {

		curX := float64(s.Left)
		lineStart := s.StartTime
		lineEnd := s.EndTime

		charN := utils.LenString(s.Text)

		// For syls of one char
		if charN == 1 || charN == 0 {
			dur = s.Duration
		} else {
			dur = int(s.Duration / charN)
		}

		for i, c := range s.Text {
			text := string(c)

			start = lineStart
			lineStart += dur

			width, _ := utils.MeasureString(d.fontFace, text)
			width *= d.Style.Scale[0] / 100.0
			// width += 2.0*s.Style.Bord + s.Style.Shadow
			middlewidth := float64(width) / 2.0

			cleft := float64(curX)
			ccenter := cleft + float64(middlewidth)
			cright := cleft + float64(width)

			align := d.Style.Alignment

			switch align {
			case 1, 4, 7: // left
				x = cleft
			case 2, 5, 8: // center
				x = ccenter
			case 3, 6, 9: // right
				x = cright
			}

			if i == charN-1 {
				// Ensure that the end time and the width of the last char
				// is the same that the end time and width of the syl
				end = lineEnd
			} else {
				end = lineStart
			}

			c := &Char{
				Dialog: Dialog{
					Layer:     d.Layer,
					Style:     d.Style,
					StyleName: d.StyleName,
					Actor:     d.Actor,
					Effect:    d.Effect,
					Tags:      d.Tags,
					Comment:   d.Comment,
					StartTime: start,
					EndTime:   end,
					Duration:  dur,
					MidTime:   end - start,
					Text:      text,
					Width:     float64(width),
					Height:    d.Height, //+ 2.0*d.Style.Bord + d.Style.Shadow,
					Size:      [2]float64{float64(width), d.Height},
					X:         float64(x),
					Y:         float64(d.Y),
					Top:       float64(d.Top),
					Middle:    float64(d.Middle),
					Bottom:    float64(d.Bottom),
					Left:      float64(cleft),
					Center:    float64(ccenter),
					Right:     float64(cright),
				},
				Inline:        s.Inline,
				SylStartTime:  s.StartTime,
				SylEndTime:    s.EndTime,
				SylMidEndTime: s.MidTime,
				SylDuration:   s.Duration,
			}

			chars = append(chars, c)

			curX += width
		}
	}
	return chars

}

// Syls list all syllables in a Line
func (d *Line) Syls() (syls []*Syl) {

	lineStart := d.StartTime
	lineEnd := d.EndTime
	end := 0
	fontFace := d.fontFace

	spaceWidth, _ := utils.MeasureString(fontFace, " ")
	spaceWidth *= d.Style.Scale[0] / 100.0
	//spaceWidth += 2*d.Style.Bord + d.Style.Shadow

	curX := d.Left + float64(d.SylN)*-d.XFix/2.0
	maxWidth := 0.0
	sumHeight := 0.0
	resx, resy := float64(d.resolution[0]), float64(d.resolution[1])

	for i, dlg := range d.syls {
		duration, inline, text := dlg[1], dlg[2], dlg[3]
		dur := utils.Str2int(duration) * 10 // cs to ms

		// Absolute times
		start := lineStart
		lineStart += dur
		if i == d.SylN-1 {
			// Ensure that the end time and the width of the last syl
			// is the same that the end time and width of the line
			end = lineEnd
		} else {
			end = lineStart
		}

		strippedText, preSpace, postSpace := utils.TrimSpaceCount(text)

		width, _ := utils.MeasureString(fontFace, strippedText)
		width *= d.Style.Scale[0] / 100.0
		//width += 2.0*d.Style.Bord + d.Style.Shadow
		height := d.Height

		middleheight := float64(height) / 2.0
		middlewidth := float64(width) / 2.0
		align := d.Style.Alignment

		curX += float64(preSpace) * spaceWidth
		sleft := float64(curX)
		scenter := sleft + middlewidth
		sright := sleft + width
		x := 0.0
		y := 0.0
		stop := 0.0
		smid := 0.0
		sbot := 0.0

		maxWidth = math.Max(maxWidth, width)
		sumHeight += height

		// line x
		if align > 6 || align < 4 {
			switch align {
			case 1, 7: // left
				x = sleft
			case 2, 8: // center
				x = scenter
			case 3, 9: // right
				x = sright
			}
			curX += width + float64(postSpace)*spaceWidth +
				float64(d.Style.Spacing) + d.XFix

		} else { // vertical alignment
			xFix := (maxWidth - width) / 2.0
			switch align {
			case 4: // left
				sleft = d.Left + xFix
				scenter = sleft + middlewidth
				sright = sleft + width
				x = sleft
			case 5: // center
				sleft = resx/2.0 - middlewidth
				scenter = sleft + middlewidth
				sright = sleft + width
				x = scenter
			case 6: // right
				sleft = d.Right - width - xFix
				scenter = sleft + middlewidth
				sright = sleft + width
				x = sright
			}
		}

		curY := resy/2.0 - sumHeight/2.0 + float64(d.Style.Spacing)

		// line y
		if align > 6 || align < 4 {
			stop = d.Top
			smid = d.Middle
			sbot = d.Bottom
			y = d.Y
		} else { // vertical alignment
			stop = curY
			smid = stop + middleheight
			sbot = stop + height
			y = smid
			// curY += height
		}

		if text != "" {
			s := &Syl{
				Dialog: Dialog{
					Layer:     d.Layer,
					Style:     d.Style,
					StyleName: d.StyleName,
					Actor:     d.Actor,
					Effect:    d.Effect,
					Tags:      d.Tags,
					Comment:   d.Comment,
					StartTime: start,
					EndTime:   end,
					Duration:  dur,
					MidTime:   end - start,
					Text:      strippedText,
					Width:     float64(width),
					Height:    float64(height),
					Size:      [2]float64{float64(width), float64(height)},
					X:         float64(x),
					Y:         float64(y),
					Top:       float64(stop),
					Middle:    float64(smid),
					Bottom:    float64(sbot),
					Left:      float64(sleft),
					Center:    float64(scenter),
					Right:     float64(sright),
				},
				Inline: inline,
			}

			syls = append(syls, s)
		}
	}
	return syls
}

// Script represent the SSA Script
type Script struct {
	Resolution         [2]int // WIDTH, HEIGHT
	VideoPath          string
	VideoZoom          float64
	VideoPosition      int
	VideoAR            float64
	MetaFilename       string
	MetaTitle          string
	MetaOriginalScript string
	MetaTranslation    string
	MetaTiming         string
	Audio              string
	LineN              int
	Shift              int
	XFix               float64
	scriptIn           *reader.Script
	scriptOut          *writer.Script
	fontFace           map[string]font.Face
}

// Lines List all the lines in a Script
func (fx *Script) Lines() (dialogs []*Line) {

	resx, resy := float64(fx.Resolution[0]), float64(fx.Resolution[1])

	for _, dlg := range fx.scriptIn.Dialog.NotCommented() {

		end := asstime.SSAtoMS(dlg.EndTime)
		start := asstime.SSAtoMS(dlg.StartTime)
		duration := end - start
		text := StripSSATags(dlg.Text)
		fontFace := fx.fontFace[dlg.StyleName]
		width, height := utils.MeasureString(fontFace, text)
		width *= dlg.Style.Scale[0] / 100.0
		//width += 2*dlg.Style.Bord + dlg.Style.Shadow
		height *= dlg.Style.Scale[1] / 100.0

		align := dlg.Style.Alignment
		ml, mr, mv := float64(dlg.Style.Margin[0]),
			float64(dlg.Style.Margin[1]),
			float64(dlg.Style.Margin[2])

		// Alignment
		middleheight := float64(height) / 2.0
		middlewidth := float64(width) / 2.0

		x := 0.0
		y := 0.0
		ltop := 0.0
		lmid := 0.0
		lbot := 0.0
		lleft := 0.0
		lcenter := 0.0
		lright := 0.0

		// line x
		switch align {
		case 1, 4, 7: // left
			lleft = ml
			lcenter = lleft + middlewidth
			lright = lleft + width
			x = lleft
		case 2, 5, 8: // center
			lleft = resx/2.0 - middlewidth
			lcenter = lleft + middlewidth
			lright = lleft + width
			x = lcenter
		case 3, 6, 9: // right
			lleft = resx - mr - width
			lcenter = lleft + middlewidth
			lright = lleft + width
			x = lright
		}

		// line y
		switch align {
		case 7, 8, 9: // top
			ltop = mv
			lmid = ltop + middleheight
			lbot = ltop + height
			y = ltop
		case 4, 5, 6: // middle
			lmid = resy / 2
			ltop = lmid - middleheight
			lbot = lmid + middleheight
			y = lmid
		case 1, 2, 3: // bottom
			lbot = resy - mv
			lmid = lbot - middleheight
			ltop = lbot - height
			y = lbot
		}

		syls := GetSyls(dlg.Text)

		charN := 0
		for _, s := range syls {
			syltext := strings.TrimSpace(s[3])
			if syltext != "" {
				for range syltext {
					charN++
				}
			}
		}

		d := &Line{
			Dialog: Dialog{
				Layer:     dlg.Layer,
				StartTime: start,
				EndTime:   end,
				Duration:  duration,
				MidTime:   start + int(duration/2),
				Style:     dlg.Style,
				StyleName: dlg.StyleName,
				Actor:     dlg.Actor,
				Effect:    dlg.Effect,
				Text:      text,
				Tags:      dlg.Tags,
				Comment:   dlg.Comment,
				Width:     float64(width),
				Height:    float64(height),
				Size:      [2]float64{float64(width), float64(height)},
				XFix:      fx.XFix,
				X:         float64(x),
				Y:         float64(y),
				Top:       float64(ltop),
				Middle:    float64(lmid),
				Bottom:    float64(lbot),
				Left:      float64(lleft),
				Center:    float64(lcenter),
				Right:     float64(lright),
			},
			Kara:       dlg.Text,
			SylN:       len(syls),
			CharN:      charN,
			syls:       syls,
			fontFace:   fontFace,
			resolution: fx.Resolution,
		}
		dialogs = append(dialogs, d)
	}
	return dialogs
}

// GetStyle get a style corresponding to name provided
func (fx *Script) GetStyle(name string) (*reader.Style, bool) {
	style, ok := fx.Styles()[name]
	return style, ok
}

// Styles list styles use the Script
func (fx *Script) Styles() map[string]*reader.Style {
	return fx.scriptIn.StyleUsed
}

// AddStyle append a Style to Script
func (fx *Script) AddStyle(sty *writer.Style) {
	fx.scriptIn.StyleUsed[sty.Name] = reader.NewStyle(sty.Name)
	fx.scriptOut.AddStyle(sty)
}

// CopyLine create a copy of the current Line
func (fx *Script) CopyLine(dialog *Line) Line {
	return *dialog
}

// CopySyl create a copy of the current Syl
func (fx *Script) CopySyl(dialog *Syl) Syl {
	return *dialog
}

// CopyChar create a copy of the current Char
func (fx *Script) CopyChar(dialog *Char) Char {
	return *dialog
}

// Add append a Dialog (Syl, Char, Line) to Script
func (fx *Script) Add(dialog interface{}) {

	switch dlg := dialog.(type) {
	case Line:
		d := NewDialog(dlg.Text)
		d.Layer = dlg.Layer
		d.Start = asstime.MStoSSA(dlg.StartTime + fx.Shift)
		d.End = asstime.MStoSSA(dlg.EndTime + fx.Shift)
		d.StyleName = dlg.StyleName
		d.Actor = dlg.Actor
		d.Effect = dlg.Effect
		d.Tags = dlg.Tags
		d.Comment = dlg.Comment
		fx.scriptOut.AddDialog(d)
	case Syl:
		d := NewDialog(dlg.Text)
		d.Layer = dlg.Layer
		d.Start = asstime.MStoSSA(dlg.StartTime + fx.Shift)
		d.End = asstime.MStoSSA(dlg.EndTime + fx.Shift)
		d.StyleName = dlg.StyleName
		d.Actor = dlg.Actor
		d.Effect = dlg.Effect
		d.Tags = dlg.Tags
		d.Comment = dlg.Comment
		fx.scriptOut.AddDialog(d)
	case Char:
		d := NewDialog(dlg.Text)
		d.Layer = dlg.Layer
		d.Start = asstime.MStoSSA(dlg.StartTime + fx.Shift)
		d.End = asstime.MStoSSA(dlg.EndTime + fx.Shift)
		d.StyleName = dlg.StyleName
		d.Actor = dlg.Actor
		d.Effect = dlg.Effect
		d.Tags = dlg.Tags
		d.Comment = dlg.Comment
		fx.scriptOut.AddDialog(d)
	default:
		fmt.Println("Not admitted object")
	}

}

// Save create the final script file (.ass)
func (fx *Script) Save(fn string) {
	fx.scriptOut.Resolution = fx.Resolution
	fx.scriptOut.VideoPath = fx.VideoPath
	fx.scriptOut.VideoZoom = fx.VideoZoom
	fx.scriptOut.VideoPosition = fx.VideoPosition
	fx.scriptOut.VideoAR = fx.VideoAR
	fx.scriptOut.MetaFilename = fx.MetaFilename
	fx.scriptOut.MetaTitle = fx.MetaTitle
	fx.scriptOut.MetaOriginalScript = fx.MetaOriginalScript
	fx.scriptOut.MetaTranslation = fx.MetaTranslation
	fx.scriptOut.MetaTiming = fx.MetaTiming
	fx.scriptOut.Audio = fx.Audio
	fx.scriptOut.Save(fn)
}

// NewEffect create a new script
func NewEffect(inFN string) *Script {
	input := reader.Read(inFN)
	output := writer.NewScript()

	fontFace := make(map[string]font.Face)

	ssampling := 1

	for _, style := range input.StyleUsed {
		s := NewStyle(style.Name)
		s.Name = style.Name
		s.FontName = style.FontName
		s.FontSize = style.FontSize
		s.Color = style.Color
		s.Bold = style.Bold
		s.Italic = style.Italic
		s.Scale = style.Scale
		s.Angle = style.Angle
		s.Bord = style.Bord
		s.Shadow = style.Shadow
		s.Alignment = style.Alignment
		s.Margin = style.Margin
		output.AddStyle(s)

		ff, err := utils.LoadFont(s.FontName, s.FontSize*ssampling)
		if err != nil {
			panic(err)
		}
		fontFace[s.Name] = ff
	}

	// Add the original karaoke commented by default in the script
	// This help to jump to the wanted line in the preview in Aegisub,
	// and/or keep a backup of the timed subs
	dok := NewDialog("### Original Karaoke ###")
	dok.Comment = true
	output.AddDialog(dok)
	LineN := 0
	for _, dlg := range input.Dialog {
		d := writer.NewDialog(dlg.Text)
		d.Layer = dlg.Layer
		d.Start = dlg.StartTime
		d.End = dlg.EndTime
		d.StyleName = dlg.StyleName
		d.Actor = dlg.Actor
		d.Effect = dlg.Effect
		d.Comment = true
		output.AddDialog(d)
		LineN++
	}
	dke := NewDialog("### Karaoke Effect ###")
	dke.Comment = true
	output.AddDialog(dke)

	return &Script{
		Resolution:         input.Resolution,
		VideoPath:          input.VideoPath,
		VideoZoom:          input.VideoZoom,
		VideoPosition:      input.VideoPosition,
		VideoAR:            input.VideoAR,
		MetaFilename:       input.MetaFilename,
		MetaTitle:          input.MetaTitle,
		MetaOriginalScript: input.MetaOriginalScript,
		MetaTranslation:    input.MetaTranslation,
		MetaTiming:         input.MetaTiming,
		Audio:              input.Audio,
		LineN:              LineN,
		fontFace:           fontFace,
		scriptIn:           input,
		scriptOut:          output,
	}
}

// NewStyle create a new Style
func NewStyle(name string) *writer.Style {
	return writer.NewStyle(name)
}

// NewDialog create a new Dialog
func NewDialog(text string) *writer.Dialog {
	return writer.NewDialog(text)
}
