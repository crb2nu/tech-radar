package techradar

import "embed"

// UI contains the static frontend assets.
//
//go:embed web/dist/*
var UI embed.FS
