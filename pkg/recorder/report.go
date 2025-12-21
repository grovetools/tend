package recorder

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"io"
	"strings"
	"text/template"

	"github.com/ActiveState/vt10x"
)

//go:embed report.html.tpl
var reportTemplate string

type reportData struct {
	Frames []htmlFrame
}

type htmlFrame struct {
	Timestamp string
	Input     string
	Snapshot  string
}

// GenerateHTMLReport creates an interactive HTML report from the recorded frames.
func GenerateHTMLReport(frames []Frame, out io.Writer) error {
	state := &vt10x.State{}
	// Create a VT instance with a dummy reader and writer
	// We'll be feeding it data via Write() calls
	vt, err := vt10x.New(state, &bytes.Buffer{}, io.Discard)
	if err != nil {
		return fmt.Errorf("failed to create terminal emulator: %w", err)
	}

	var htmlFrames []htmlFrame
	for _, frame := range frames {
		// Feed the raw ANSI output to the terminal emulator
		vt.Write([]byte(frame.Output))

		// Render the state of the emulator to HTML
		snapshotHTML := renderEmulatorToHTML(state)

		htmlFrames = append(htmlFrames, htmlFrame{
			Timestamp: fmt.Sprintf("+%.3fs", frame.Timestamp.Seconds()),
			Input:     formatInput(frame.Input),
			Snapshot:  snapshotHTML,
		})
	}

	tpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse report template: %w", err)
	}

	return tpl.Execute(out, reportData{Frames: htmlFrames})
}

// renderEmulatorToHTML converts the terminal emulator's grid into styled HTML.
func renderEmulatorToHTML(state *vt10x.State) string {
	var sb strings.Builder
	rows, cols := state.Size()

	for r := 0; r < rows; r++ {
		var currentFG, currentBG vt10x.Color
		var currentStyle string

		for c := 0; c < cols; c++ {
			ch, fg, bg := state.Cell(c, r)

			// Check if style changed
			if fg != currentFG || bg != currentBG {
				if currentStyle != "" {
					sb.WriteString("</span>")
				}
				currentFG = fg
				currentBG = bg
				currentStyle = cellStyleToHTML(fg, bg)
				sb.WriteString(currentStyle)
			}

			sb.WriteString(html.EscapeString(string(ch)))
		}

		if currentStyle != "" {
			sb.WriteString("</span>")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// cellStyleToHTML converts colors to an opening <span> tag with inline styles.
func cellStyleToHTML(fg, bg vt10x.Color) string {
	var styles []string

	// Convert foreground color
	if fg != vt10x.DefaultFG {
		styles = append(styles, fmt.Sprintf("color: %s;", colorToCSS(fg)))
	}

	// Convert background color
	if bg != vt10x.DefaultBG {
		styles = append(styles, fmt.Sprintf("background-color: %s;", colorToCSS(bg)))
	}

	if len(styles) == 0 {
		return "<span>"
	}
	return fmt.Sprintf(`<span style="%s">`, strings.Join(styles, " "))
}

// colorToCSS converts a vt10x.Color to a CSS color string.
func colorToCSS(c vt10x.Color) string {
	// Basic 8 colors (0-7) and bright colors (8-15)
	colors := []string{
		"#000000", // 0: Black
		"#cd0000", // 1: Red
		"#00cd00", // 2: Green
		"#cdcd00", // 3: Yellow
		"#0000ee", // 4: Blue
		"#cd00cd", // 5: Magenta
		"#00cdcd", // 6: Cyan
		"#e5e5e5", // 7: White
		"#7f7f7f", // 8: Bright Black (Gray)
		"#ff0000", // 9: Bright Red
		"#00ff00", // 10: Bright Green
		"#ffff00", // 11: Bright Yellow
		"#5c5cff", // 12: Bright Blue
		"#ff00ff", // 13: Bright Magenta
		"#00ffff", // 14: Bright Cyan
		"#ffffff", // 15: Bright White
	}

	// Convert color index to int
	idx := int(c)

	// Handle basic 16 colors
	if idx >= 0 && idx < len(colors) {
		return colors[idx]
	}

	// For extended colors (16-255), use a simple grayscale approximation
	// This is a simplification; a full implementation would use the xterm 256 color palette
	if idx >= 16 && idx < 256 {
		// Simplified: map to grayscale
		val := (idx - 16) * 255 / 239
		return fmt.Sprintf("rgb(%d,%d,%d)", val, val, val)
	}

	// Default color
	return "#d4d4d4"
}

// formatInput makes control characters visible for the report.
func formatInput(input string) string {
	input = strings.ReplaceAll(input, "\r", "<Enter>")
	input = strings.ReplaceAll(input, "\x1b", "<Esc>")
	// Add more replacements for other control characters as needed
	return html.EscapeString(input)
}
