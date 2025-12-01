// Package cli provides command-line interface commands for the seeder.
package cli

import "errors"

// Sentinel errors for CLI validation
var (
	// errNoSourceProvided is returned when no torrent source is specified in the add command
	errNoSourceProvided = errors.New("must specify one of --file, --magnet, or --infohash")

	// errMultipleSourcesProvided is returned when more than one torrent source is specified
	errMultipleSourcesProvided = errors.New("specify only one of --file, --magnet, or --infohash")

	// errInfoHashRequired is returned when the infohash flag is missing for commands that require it
	errInfoHashRequired = errors.New("--infohash is required")
)
