package config

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

// StyleListener represents a skin's listener.
type StyleListener interface {
	// StylesChanged notifies listener the skin changed.
	StylesChanged(*Styles)
}

type (

	// Styles tracks process compose styling options.
	Styles struct {
		Style Style `yaml:"style"`
	}

	// Style tracks PC styles.
	Style struct {
		Name      string    `yaml:"name"`
		Body      Body      `yaml:"body"`
		StatTable StatTable `yaml:"stat_table"`
		ProcTable ProcTable `yaml:"proc_table"`
		Help      Help      `yaml:"help"`
		Dialog    Dialog    `yaml:"dialog"`
	}

	// Body tracks body styles.
	Body struct {
		FgColor            Color `yaml:"fgColor"`
		BgColor            Color `yaml:"bgColor"`
		SecondaryTextColor Color `yaml:"secondaryTextColor"`
		TertiaryTextColor  Color `yaml:"tertiaryTextColor"`
		BorderColor        Color `yaml:"borderColor"`
	}

	// StatTable tracks stats table styles.
	StatTable struct {
		KeyFgColor   Color `yaml:"keyFgColor"`
		ValueFgColor Color `yaml:"valueFgColor"`
		BgColor      Color `yaml:"bgColor"`
		LogoColor    Color `yaml:"logoColor"`
	}

	// ProcTable tracks processes table styles.
	ProcTable struct {
		FgColor       Color `yaml:"fgColor"`
		FgWarning     Color `yaml:"fgWarning"`
		FgPending     Color `yaml:"fgPending"`
		FgCompleted   Color `yaml:"fgCompleted"`
		FgError       Color `yaml:"fgError"`
		BgColor       Color `yaml:"bgColor"`
		HeaderFgColor Color `yaml:"headerFgColor"`
	}

	// Help tracks help styles.
	Help struct {
		KeyColor        Color `yaml:"keyColor"`
		FgColor         Color `yaml:"fgColor"`
		HlColor         Color `yaml:"hlColor"`
		FgCategoryColor Color `yaml:"categoryFgColor"`
	}

	// Dialog tracks dialog styles.
	Dialog struct {
		FgColor            Color `yaml:"fgColor"`
		BgColor            Color `yaml:"bgColor"`
		ButtonFgColor      Color `yaml:"buttonFgColor"`
		ButtonBgColor      Color `yaml:"buttonBgColor"`
		ButtonFocusFgColor Color `yaml:"buttonFocusFgColor"`
		ButtonFocusBgColor Color `yaml:"buttonFocusBgColor"`
		LabelFgColor       Color `yaml:"labelFgColor"`
		FieldFgColor       Color `yaml:"fieldFgColor"`
		FieldBgColor       Color `yaml:"fieldBgColor"`
	}
)

func newStyle() Style {
	return Style{
		Name:      "Default",
		Body:      newBody(),
		StatTable: newStatTable(),
		ProcTable: newProcTable(),
		Help:      newHelp(),
		Dialog:    newDialog(),
	}
}

func newBody() Body {
	return Body{
		FgColor:            "white",
		BgColor:            "black",
		SecondaryTextColor: "yellow",
		TertiaryTextColor:  "green",
		BorderColor:        "white",
	}
}

func newStatTable() StatTable {
	return StatTable{
		KeyFgColor:   "yellow",
		ValueFgColor: "white",
		BgColor:      "black",
		LogoColor:    "yellow",
	}
}

func newProcTable() ProcTable {
	return ProcTable{
		HeaderFgColor: "white",
		FgColor:       "lightskyblue",
		BgColor:       "black",
		FgWarning:     "yellow",
		FgPending:     "grey",
		FgCompleted:   "lightgreen",
		FgError:       "red",
	}
}

func newHelp() Help {
	return Help{
		FgColor:         "black",
		KeyColor:        "white",
		HlColor:         "green",
		FgCategoryColor: "lightskyblue",
	}
}

func newDialog() Dialog {
	return Dialog{
		FgColor:            "cadetblue",
		BgColor:            "black",
		ButtonBgColor:      "lightskyblue",
		ButtonFgColor:      "black",
		ButtonFocusBgColor: "dodgerblue",
		ButtonFocusFgColor: "black",
		LabelFgColor:       "yellow",
		FieldFgColor:       "black",
		FieldBgColor:       "lightskyblue",
	}
}

// NewStyles creates a new default config.
func NewStyles() *Styles {
	//var s Styles
	//if err := yaml.Unmarshal(stockThemeTpl, &s); err == nil {
	//	return &s
	//}

	return &Styles{
		Style: newStyle(),
	}
}

// Load process compose styles from file.
func (s *Styles) Load(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, s); err != nil {
		return err
	}

	return nil
}

// FgColor returns the foreground color.
func (s *Styles) FgColor() tcell.Color {
	return s.Body().FgColor.Color()
}

// BgColor returns the background color.
func (s *Styles) BgColor() tcell.Color {
	return s.Body().BgColor.Color()
}

// BorderColor returns the border color.
func (s *Styles) BorderColor() tcell.Color {
	return s.Body().BorderColor.Color()
}

// Body returns body styles.
func (s *Styles) Body() Body {
	return s.Style.Body
}

// StatTable returns stat table styles.
func (s *Styles) StatTable() StatTable {
	return s.Style.StatTable
}

// ProcTable returns process table styles.
func (s *Styles) ProcTable() ProcTable {
	return s.Style.ProcTable
}

// Help returns help styles.
func (s *Styles) Help() Help {
	return s.Style.Help
}

// Dialog returns dialog styles.
func (s *Styles) Dialog() Dialog {
	return s.Style.Dialog
}

// Update apply terminal colors based on styles.
func (s *Styles) Update() {
	tview.Styles.PrimitiveBackgroundColor = s.BgColor()
	//tview.Styles.ContrastBackgroundColor = s.BgColor()
	tview.Styles.MoreContrastBackgroundColor = s.BgColor()
	tview.Styles.PrimaryTextColor = s.FgColor()
	tview.Styles.BorderColor = s.BorderColor()
	tview.Styles.TitleColor = s.FgColor()
	tview.Styles.GraphicsColor = s.FgColor()
	tview.Styles.SecondaryTextColor = s.Body().SecondaryTextColor.Color()
	tview.Styles.TertiaryTextColor = s.Body().TertiaryTextColor.Color()
	tview.Styles.InverseTextColor = s.FgColor()
	tview.Styles.ContrastSecondaryTextColor = s.FgColor()
}

// GetStyleName returns the style name
func (s *Styles) GetStyleName() string {
	return s.Style.Name
}

// Dump for debug.
func (s *Styles) Dump(w io.Writer) {
	b, _ := yaml.Marshal(s)
	_, _ = fmt.Fprintf(w, "%s", b)
}
