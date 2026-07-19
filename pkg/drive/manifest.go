package drive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/grovetools/tend/pkg/tui"
)

// ManifestSchemaVersion identifies the evidence-bundle schema. Consumers should
// gate on this string before trusting the shape.
const ManifestSchemaVersion = "tend.drive/v1"

// ManifestFileName is the fixed name of the manifest within the bundle.
const ManifestFileName = "manifest.json"

// Manifest is the schema-versioned root of an evidence bundle. Every documented
// key is always present; zero values are emitted rather than omitted so that
// consumers never have to distinguish "absent" from "empty".
type Manifest struct {
	SchemaVersion string         `json:"schema_version"`
	Socket        string         `json:"socket"`
	Session       string         `json:"session"`
	Mode          string         `json:"mode"`
	Script        string         `json:"script"`
	StartedAt     time.Time      `json:"started_at"`
	EndedAt       time.Time      `json:"ended_at"`
	ExitCode      int            `json:"exit_code"`
	FailedStep    int            `json:"failed_step"`
	Steps         []ManifestStep `json:"steps"`
}

// ManifestStep is one step's record within the manifest.
type ManifestStep struct {
	Index     int       `json:"index"`
	Kind      string    `json:"kind"`
	Arg       string    `json:"arg"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Outcome   string    `json:"outcome"`
	Failure   string    `json:"failure"`
	Files     []string  `json:"files"`
}

// ManifestMeta carries the attach parameters recorded alongside the run.
type ManifestMeta struct {
	Socket  string
	Session string
	Mode    string
	Script  string
}

// BuildManifest assembles a Manifest from the run result and attach metadata.
// Nil slices are normalized to empty slices so the JSON always has arrays.
func BuildManifest(meta ManifestMeta, result *RunResult) Manifest {
	steps := make([]ManifestStep, 0, len(result.Steps))
	for _, s := range result.Steps {
		files := s.Files
		if files == nil {
			files = []string{}
		}
		steps = append(steps, ManifestStep{
			Index:     s.Index,
			Kind:      string(s.Kind),
			Arg:       s.Arg,
			StartedAt: s.StartedAt,
			EndedAt:   s.EndedAt,
			Outcome:   string(s.Outcome),
			Failure:   s.Failure,
			Files:     files,
		})
	}
	return Manifest{
		SchemaVersion: ManifestSchemaVersion,
		Socket:        meta.Socket,
		Session:       meta.Session,
		Mode:          meta.Mode,
		Script:        meta.Script,
		StartedAt:     result.StartedAt,
		EndedAt:       result.EndedAt,
		ExitCode:      result.ExitCode(),
		FailedStep:    result.FailedIndex,
		Steps:         steps,
	}
}

// WriteManifest writes the manifest as pretty JSON into dir/manifest.json.
func WriteManifest(dir string, m Manifest) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, ManifestFileName), data, 0o644)
}

// marshalSnapshot renders a debug snapshot as pretty JSON for <label>.json.
func marshalSnapshot(snap *tui.DebugSnapshot) ([]byte, error) {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
