package main

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/thepeacockproject/Octopussy/internal/content"
)

const unlockablesPath string = "data/unlockables.json"

func main() {
	raw, err := os.ReadFile(unlockablesPath)
	if err != nil {
		panic(err)
	}

	// Strip (potential) UTF-8 BOM
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))

	var entries []content.Unlockable
	if err := json.Unmarshal(raw, &entries); err != nil {
		panic(err)
	}

	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		panic(err)
	}

	out = append(out, '\n')

	if err := os.WriteFile(unlockablesPath, out, 0o644); err != nil {
		panic(err)
	}
}
