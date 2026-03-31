// Package theme handles loading and watching the Omarchy color theme.
package theme

import (
	"github.com/pelletier/go-toml/v2"
	"os"
)

// ThemeColors holds all color tokens read from an Omarchy colors.toml file.
type ThemeColors struct {
	Background          string `toml:"background"           json:"background"`
	Foreground          string `toml:"foreground"           json:"foreground"`
	Accent              string `toml:"accent"               json:"accent"`
	Cursor              string `toml:"cursor"               json:"cursor"`
	SelectionBackground string `toml:"selection_background" json:"selectionBackground"`
	SelectionForeground string `toml:"selection_foreground" json:"selectionForeground"`
	Color0              string `toml:"color0"               json:"color0"`
	Color1              string `toml:"color1"               json:"color1"`
	Color2              string `toml:"color2"               json:"color2"`
	Color3              string `toml:"color3"               json:"color3"`
	Color4              string `toml:"color4"               json:"color4"`
	Color5              string `toml:"color5"               json:"color5"`
	Color6              string `toml:"color6"               json:"color6"`
	Color7              string `toml:"color7"               json:"color7"`
	Color8              string `toml:"color8"               json:"color8"`
	Color9              string `toml:"color9"               json:"color9"`
	Color10             string `toml:"color10"              json:"color10"`
	Color11             string `toml:"color11"              json:"color11"`
	Color12             string `toml:"color12"              json:"color12"`
	Color13             string `toml:"color13"              json:"color13"`
	Color14             string `toml:"color14"              json:"color14"`
	Color15             string `toml:"color15"              json:"color15"`
}

// Load reads and parses an Omarchy colors.toml file at the given path.
func Load(path string) (ThemeColors, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ThemeColors{}, err
	}

	var colors ThemeColors
	if err := toml.Unmarshal(data, &colors); err != nil {
		return ThemeColors{}, err
	}

	return colors, nil
}
