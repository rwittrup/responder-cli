// Package calltypes holds the built-in Audio Demo call types the CLI
// can start. The data is copied from the prepared911 dispatch UI's
// built-in demo list (turbo/apps/dispatch/src/modules/demo/
// demoAudioFiles.ts) and must be kept in sync by hand. Custom
// (database-backed) demos are not supported.
package calltypes

import (
	"fmt"
	"sort"
	"strings"
)

// CallType is a built-in Audio Demo scenario.
type CallType struct {
	Name                   string
	CallerAudioURL         string
	DispatcherAudioURL     string
	LanguageCode           string
	CallerLanguageCode     string
	DispatcherLanguageCode string
}

// Builtins are the always-on built-in demos from the dispatch UI.
var Builtins = []CallType{
	{
		Name:               "Shooting Incident - English",
		CallerAudioURL:     "https://static.cdn.prepared911.dev/audio-demos/shooting-incident-english-caller.raw",
		DispatcherAudioURL: "https://static.cdn.prepared911.dev/audio-demos/shooting-incident-english-dispatcher.raw",
		LanguageCode:       "en-US",
	},
	{
		Name:               "Medical Emergency - School",
		CallerAudioURL:     "https://static.cdn.prepared911.dev/audio-demos/shooting-incident-english-caller.raw",
		DispatcherAudioURL: "https://static.cdn.prepared911.dev/audio-demos/shooting-incident-english-dispatcher.raw",
		LanguageCode:       "en-US",
	},
	{
		Name:               "House Fire - English",
		CallerAudioURL:     "https://static.cdn.prepared911.dev/audio-demos/house-fire-english-caller.raw",
		DispatcherAudioURL: "https://static.cdn.prepared911.dev/audio-demos/house-fire-english-dispatcher.raw",
		LanguageCode:       "en-US",
	},
	{
		Name:               "Home Invasion - English",
		CallerAudioURL:     "https://static.cdn.prepared911.dev/audio-demos/home-invasion-english-caller.raw",
		DispatcherAudioURL: "https://static.cdn.prepared911.dev/audio-demos/home-invasion-english-dispatcher.raw",
		LanguageCode:       "en-US",
	},
	{
		Name:                   "Shooting Incident - Vietnamese",
		CallerAudioURL:         "https://static.cdn.prepared911.dev/audio-demos/vietnamese-shooting-caller-take1-trimmed.raw",
		DispatcherAudioURL:     "https://static.cdn.prepared911.dev/audio-demos/vietnamese-shooting-dispatcher-take1.raw",
		LanguageCode:           "vi",
		CallerLanguageCode:     "vi",
		DispatcherLanguageCode: "en-US",
	},
	{
		Name:                   "Prepared Translator - Spanish",
		CallerAudioURL:         "https://static.cdn.prepared911.dev/audio-demos/spanish-translator-caller.raw",
		DispatcherAudioURL:     "https://static.cdn.prepared911.dev/audio-demos/spanish-translator-dispatcher.raw",
		LanguageCode:           "es",
		CallerLanguageCode:     "es",
		DispatcherLanguageCode: "en-US",
	},
	{
		Name:               "Crime In-Progress - Mayo Blvd",
		CallerAudioURL:     "https://static.cdn.prepared911.dev/audio-demos/axon-hq-gsoc-caller.raw",
		DispatcherAudioURL: "https://static.cdn.prepared911.dev/audio-demos/axon-hq-gsoc-dispatcher.raw",
		LanguageCode:       "en-US",
	},
	{
		Name:               "Axon Arena",
		CallerAudioURL:     "https://static.cdn.prepared911.dev/audio-demos/axon-arena-caller.raw",
		DispatcherAudioURL: "https://static.cdn.prepared911.dev/audio-demos/axon-arena-dispatcher.raw",
		LanguageCode:       "en-US",
	},
}

// DefaultName is the call type the portal pre-selects.
const DefaultName = "Shooting Incident - English"

// Lookup returns the call type with the given name.
func Lookup(name string) (CallType, error) {
	for _, ct := range Builtins {
		if ct.Name == name {
			return ct, nil
		}
	}
	return CallType{}, fmt.Errorf("unknown call type %q (valid call types: %s)", name, Names())
}

// Names lists the valid call type names, sorted.
func Names() string {
	names := make([]string, 0, len(Builtins))
	for _, ct := range Builtins {
		names = append(names, ct.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
