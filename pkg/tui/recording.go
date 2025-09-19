package tui

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"strings"
	"sync"
	"time"
)

// SessionRecording tracks all events and captures during a TUI session.
type SessionRecording struct {
	enabled    bool
	startTime  time.Time
	events     []RecordedEvent
	captures   []ScreenCapture
	outputPath string
	mu         sync.Mutex
}

// RecordedEvent represents a single action or assertion in the session.
type RecordedEvent struct {
	Timestamp time.Time   `json:"timestamp"`
	Type      string      `json:"type"` // "key", "capture", "assert", "wait", "navigate"
	Data      interface{} `json:"data"`
	Result    string      `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// ScreenCapture represents a snapshot of the TUI at a point in time.
type ScreenCapture struct {
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
	Raw       string    `json:"raw,omitempty"`
}

// StartRecording begins recording the session to the specified output path.
func (s *Session) StartRecording(outputPath string) error {
	s.recording = &SessionRecording{
		enabled:    true,
		startTime:  time.Now(),
		events:     []RecordedEvent{},
		captures:   []ScreenCapture{},
		outputPath: outputPath,
	}

	// Capture initial state
	content, _ := s.Capture(WithRawOutput())
	clean, _ := s.Capture(WithCleanedOutput())
	s.recording.captures = append(s.recording.captures, ScreenCapture{
		Timestamp: time.Now(),
		Content:   clean,
		Raw:       content,
	})

	return nil
}

// StopRecording stops recording and saves the session data.
func (s *Session) StopRecording() error {
	if s.recording == nil {
		return fmt.Errorf("no recording in progress")
	}

	s.recording.mu.Lock()
	s.recording.enabled = false
	s.recording.mu.Unlock()

	return s.SaveRecording()
}

// SaveRecording exports the session data to multiple formats.
func (s *Session) SaveRecording() error {
	if s.recording == nil {
		return fmt.Errorf("no recording to save")
	}

	// Generate HTML report
	htmlContent := s.generateHTMLReport()
	htmlPath := s.recording.outputPath + ".html"
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to save HTML report: %w", err)
	}

	// Save JSON for programmatic access
	jsonData, err := json.MarshalIndent(s.recording, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal recording to JSON: %w", err)
	}
	jsonPath := s.recording.outputPath + ".json"
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to save JSON data: %w", err)
	}

	return nil
}

// recordEvent adds an event to the recording if recording is enabled.
func (s *Session) recordEvent(eventType string, data interface{}, result string, err error) {
	if s.recording == nil || !s.recording.enabled {
		return
	}

	s.recording.mu.Lock()
	defer s.recording.mu.Unlock()

	event := RecordedEvent{
		Timestamp: time.Now(),
		Type:      eventType,
		Data:      data,
		Result:    result,
	}

	if err != nil {
		event.Error = err.Error()
	}

	s.recording.events = append(s.recording.events, event)
}

// captureForRecording captures the current screen state for the recording.
func (s *Session) captureForRecording() {
	if s.recording == nil || !s.recording.enabled {
		return
	}

	content, _ := s.Capture(WithCleanedOutput())
	raw, _ := s.Capture(WithRawOutput())

	s.recording.mu.Lock()
	defer s.recording.mu.Unlock()

	s.recording.captures = append(s.recording.captures, ScreenCapture{
		Timestamp: time.Now(),
		Content:   content,
		Raw:       raw,
	})
}

// GetKeyHistory returns all keys sent during the session.
func (s *Session) GetKeyHistory() []string {
	if s.recording == nil {
		return []string{}
	}

	s.recording.mu.Lock()
	defer s.recording.mu.Unlock()

	var history []string
	for _, event := range s.recording.events {
		if event.Type == "key" {
			if data, ok := event.Data.(map[string]interface{}); ok {
				if keys, ok := data["keys"].([]string); ok {
					history = append(history, keys...)
				}
			}
		}
	}
	return history
}

// TakeScreenshot captures the current TUI state to a file.
func (s *Session) TakeScreenshot(filepath string) error {
	raw, err := s.Capture(WithRawOutput())
	if err != nil {
		return fmt.Errorf("failed to capture for screenshot: %w", err)
	}

	// Save raw ANSI for terminal playback
	if err := os.WriteFile(filepath, []byte(raw), 0644); err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}

	// Record the screenshot event
	s.recordEvent("screenshot", map[string]interface{}{"path": filepath}, "screenshot saved", nil)

	return nil
}

// generateHTMLReport creates an interactive HTML timeline of the session.
func (s *Session) generateHTMLReport() string {
	var sb strings.Builder

	// HTML header with styles
	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
    <title>TUI Session Recording</title>
    <meta charset="UTF-8">
    <style>
        body {
            font-family: 'Cascadia Code', 'Courier New', monospace;
            background: #1e1e1e;
            color: #d4d4d4;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        h1 {
            color: #4EC9B0;
            border-bottom: 2px solid #007acc;
            padding-bottom: 10px;
        }
        .metadata {
            background: #252526;
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 20px;
        }
        .timeline {
            margin: 20px 0;
        }
        .event {
            margin: 10px 0;
            padding: 15px;
            border-left: 3px solid #007acc;
            background: #252526;
            border-radius: 0 5px 5px 0;
            transition: background 0.2s;
        }
        .event:hover {
            background: #2d2d30;
        }
        .event-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        .event-type {
            display: inline-block;
            padding: 2px 8px;
            border-radius: 3px;
            font-size: 0.9em;
            font-weight: bold;
        }
        .type-key { background: #4EC9B0; color: #1e1e1e; }
        .type-wait { background: #569CD6; color: #fff; }
        .type-assert { background: #CE9178; color: #fff; }
        .type-navigate { background: #C586C0; color: #fff; }
        .type-screenshot { background: #608B4E; color: #fff; }
        .timestamp {
            color: #808080;
            font-size: 0.9em;
        }
        .event-data {
            font-family: 'Cascadia Code', monospace;
        }
        .keys {
            color: #DCDCAA;
            background: #383838;
            padding: 2px 6px;
            border-radius: 3px;
            margin: 0 3px;
            display: inline-block;
        }
        .error {
            color: #f48771;
            margin-top: 8px;
            padding: 8px;
            background: #3c1f1f;
            border-radius: 3px;
        }
        .success {
            color: #4EC9B0;
        }
        .screen-capture {
            margin-top: 15px;
            padding: 15px;
            background: #000;
            color: #0f0;
            border: 1px solid #333;
            border-radius: 5px;
            white-space: pre-wrap;
            word-wrap: break-word;
            font-size: 14px;
            max-height: 400px;
            overflow-y: auto;
        }
        .controls {
            position: sticky;
            top: 0;
            background: #1e1e1e;
            padding: 15px 0;
            border-bottom: 1px solid #333;
            margin-bottom: 20px;
            z-index: 100;
        }
        button {
            background: #007acc;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 5px;
            cursor: pointer;
            margin-right: 10px;
            font-size: 14px;
            transition: background 0.2s;
        }
        button:hover {
            background: #005a9e;
        }
        .hidden {
            display: none;
        }
    </style>
    <script>
        function toggleScreens() {
            const screens = document.querySelectorAll('.screen-capture');
            screens.forEach(s => s.classList.toggle('hidden'));
            const btn = document.getElementById('toggle-btn');
            btn.textContent = screens[0].classList.contains('hidden') ? 'Show Screens' : 'Hide Screens';
        }
        
        function playback() {
            const events = document.querySelectorAll('.event');
            let index = 0;
            
            // Reset all events
            events.forEach(e => e.style.opacity = '0.3');
            
            const interval = setInterval(() => {
                if (index >= events.length) {
                    clearInterval(interval);
                    events.forEach(e => e.style.opacity = '1');
                    return;
                }
                
                events[index].style.opacity = '1';
                events[index].scrollIntoView({behavior: 'smooth', block: 'center'});
                index++;
            }, 500);
        }
        
        function filterEvents(type) {
            const events = document.querySelectorAll('.event');
            events.forEach(e => {
                if (type === 'all' || e.dataset.type === type) {
                    e.style.display = 'block';
                } else {
                    e.style.display = 'none';
                }
            });
        }
    </script>
</head>
<body>
    <h1>🎬 TUI Session Recording</h1>
`)

	// Add metadata section
	duration := time.Duration(0)
	if len(s.recording.events) > 0 {
		duration = s.recording.events[len(s.recording.events)-1].Timestamp.Sub(s.recording.startTime)
	}

	sb.WriteString(fmt.Sprintf(`
    <div class="metadata">
        <strong>Recording Started:</strong> %s<br>
        <strong>Duration:</strong> %s<br>
        <strong>Total Events:</strong> %d<br>
        <strong>Total Captures:</strong> %d
    </div>
`,
		s.recording.startTime.Format(time.RFC3339),
		duration.Round(time.Millisecond).String(),
		len(s.recording.events),
		len(s.recording.captures),
	))

	// Add controls
	sb.WriteString(`
    <div class="controls">
        <button id="toggle-btn" onclick="toggleScreens()">Hide Screens</button>
        <button onclick="playback()">▶️ Playback</button>
        <button onclick="filterEvents('all')">All</button>
        <button onclick="filterEvents('key')">Keys</button>
        <button onclick="filterEvents('wait')">Waits</button>
        <button onclick="filterEvents('assert')">Asserts</button>
        <button onclick="filterEvents('navigate')">Navigation</button>
    </div>
    <div class="timeline">
`)

	// Add events
	captureIndex := 0
	for _, event := range s.recording.events {
		elapsed := event.Timestamp.Sub(s.recording.startTime)

		sb.WriteString(fmt.Sprintf(`
        <div class="event" data-type="%s">
            <div class="event-header">
                <span class="event-type type-%s">%s</span>
                <span class="timestamp">%s</span>
            </div>
            <div class="event-data">`,
			event.Type, event.Type, strings.ToUpper(event.Type),
			elapsed.Round(time.Millisecond).String()))

		// Format event data based on type
		switch event.Type {
		case "key":
			if data, ok := event.Data.(map[string]interface{}); ok {
				if keys, ok := data["keys"].([]string); ok {
					sb.WriteString("Sent keys: ")
					for _, k := range keys {
						sb.WriteString(fmt.Sprintf(`<span class="keys">%s</span>`,
							html.EscapeString(k)))
					}
				}
			}
		case "wait", "assert", "navigate":
			if event.Result != "" {
				sb.WriteString(fmt.Sprintf(`<span class="success">✓ %s</span>`,
					html.EscapeString(event.Result)))
			}
		case "screenshot":
			if data, ok := event.Data.(map[string]interface{}); ok {
				if path, ok := data["path"].(string); ok {
					sb.WriteString(fmt.Sprintf("Screenshot saved to: %s", html.EscapeString(path)))
				}
			}
		}

		// Add error if present
		if event.Error != "" {
			sb.WriteString(fmt.Sprintf(`<div class="error">✗ Error: %s</div>`,
				html.EscapeString(event.Error)))
		}

		sb.WriteString("</div>")

		// Add screen capture if available and matching timestamp
		if captureIndex < len(s.recording.captures) {
			capture := s.recording.captures[captureIndex]
			if capture.Timestamp.Sub(event.Timestamp).Abs() < 200*time.Millisecond {
				sb.WriteString(fmt.Sprintf(`
            <div class="screen-capture">%s</div>`,
					html.EscapeString(capture.Content)))
				captureIndex++
			}
		}

		sb.WriteString("</div>\n")
	}

	// Close HTML
	sb.WriteString(`
    </div>
</body>
</html>`)

	return sb.String()
}