// Package writer create SSA/ASS Subtitle Script
package writer2

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"text/template"

	"github.com/Alquimista/eyecandy/asstime"
	"github.com/Alquimista/eyecandy/color"
	"github.com/Alquimista/eyecandy/utils"
)

const dummyVideoTemplate string = "?dummy:%.6f:%d:%d:%d:%d:%d:%d%s:"
const dummyAudioTemplate string = "dummy-audio:silence?sr=44100&bd=16&" +
	"ch=1&ln=396900000:" // silence?, noise? TODO: dummy audio function
const tmpl string = "writer2/template.ass.gotmpl"

// Script SSA/ASS Subtitle Script.
type Script struct {
	Dialog             []*Dialog
	Style              map[string]*Style
	Comment            string
	Resolution         [2]int // WIDTH, HEIGHT map[string]string
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
}

// String get the generated SSA/ASS Script as a String
func (s *Script) String() string {

	if s.VideoPath == "" {
		s.VideoPath = DummyVideo(
			asstime.FpsNtscFilm,
			s.Resolution[0], s.Resolution[1],
			"#000",
			false,
			600)
	}
	if s.VideoPath != "" {
		if strings.HasPrefix(s.VideoPath, "?dummy") {
			s.Audio = dummyAudioTemplate
		} else {
			s.Audio = s.VideoPath
		}
	}

	// TODO: Write only used styles in dialog
	fm := template.FuncMap{
		"div":  func(a, b int) float64 { return float64(a) / float64(b) },
		"ssal": func(c color.Color) string { return c.SSAL() },
	}

	t := template.New("template.ass.gotmpl").Funcs(fm)

	base := "."
	if _, filename, _, ok := runtime.Caller(0); ok {
		base = path.Join(path.Dir(filename), "..")
	}

	t, err := t.ParseFiles(path.Join(base, tmpl))
	// t, err := t.Parse(tmpl)
	if err != nil {
		log.Fatal("Parse: ", err)
		return ""
	}

	var buff bytes.Buffer
	// err = t.Execute(os.Stdout, s)
	err = t.Execute(&buff, s)

	if err != nil {
		log.Fatal("Execute: ", err)
		return ""
	}

	return buff.String()
}

// Save write an SSA/ASS Subtitle Script.
func (s *Script) Save(fn string) {

	// fmt.Println(s.String())

	BOM := "\uFEFF"
	f, err := os.Create(fn)
	if err != nil {
		panic(fmt.Errorf("writer: failed saving subtitle file: %s", err))
	}
	defer f.Close()

	s.MetaFilename = fn

	n, err := f.WriteString(BOM + s.String())
	if err != nil {
		fmt.Println(n, err)
	}

	// save changes
	err = f.Sync()
}

// NewScript create a new Script Struct with defaults
func NewScript() *Script {
	return &Script{
		Comment:       "Script generated by Eyecandy",
		MetaTitle:     "Default Eyecandy file",
		VideoZoom:     0.75,
		Style:         map[string]*Style{},
		VideoPosition: 0,
	}
}

// Dialog Represent the subtitle"s lines.
type Dialog struct {
	Layer     int
	Start     string
	End       string
	StyleName string
	Actor     string
	Effect    string
	Text      string
	Tags      string
	Comment   bool
}

// AddDialog add a Dialog to SSA/ASS Script.
func (s *Script) AddDialog(d *Dialog) {
	if d.Text != "" {
		s.Dialog = append(s.Dialog, d)
	}
}

// NewDialog create a new Dialog Struct with defaults
func NewDialog(text string) *Dialog {
	return &Dialog{
		StyleName: "Default",
		Start:     "0:00:00.00", End: "0:00:05.00",
		Text: text}
}

// Style represent subtitle"s styles.
type Style struct {
	Name      string
	FontName  string
	FontSize  int
	Color     [4]*color.Color //Primary, Secondary, Bord, Shadow
	Bold      bool
	Italic    bool
	Underline bool
	StrikeOut bool
	Scale     [2]float64 // WIDTH, HEIGHT map[string]string
	Spacing   float64
	Angle     int
	OpaqueBox bool
	Bord      float64
	Shadow    float64
	Alignment int
	Margin    [3]int // L, R, V map[string]string
	Encoding  int
}

// StyleExists get if a Style exists matching the argument name
func (s *Script) styleExists(name string) bool {
	_, ok := s.Style[name]
	return ok
}

// AddStyle add a Style to SSA/ASS Script.
func (s *Script) AddStyle(sty *Style) {
	if !s.styleExists(sty.Name) {
		s.Style[sty.Name] = sty
	}
}

// NewStyle create a new Style Struct with defaults
func NewStyle(name string) *Style {
	return &Style{
		Name:     name,
		FontName: "Arial",
		FontSize: 35,
		Color: [4]*color.Color{
			color.NewFromHEX(0xFFFFFF), //Primary
			color.NewFromHEX(0x0000FF), //Secondary
			color.NewFromHEX(0x000000), //Bord
			color.NewFromHEX(0x000000), //Shadow
		},
		Scale:     [2]float64{100, 100},
		Bord:      2,
		Alignment: 8,
		Margin:    [3]int{10, 20, 10},
	}
}

// DummyVideo blank video file.
func DummyVideo(framerate float64, w, h int, hexc string, cb bool, timeS int) string {
	c := color.NewFromHTML(hexc)
	checkboard := ""
	if cb {
		checkboard = "c"
	}
	frames := asstime.MStoFrames(timeS*asstime.Second, framerate)
	return fmt.Sprintf(
		dummyVideoTemplate,
		utils.Round(framerate, 3), frames, w, h, c.R, c.G, c.B, checkboard)
}
