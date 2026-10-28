// Package handlers implements HTTP handlers and gameplay logic for the GuWorld game.
// Combat logic is split across files with the combat_ prefix. Realtime world
// actions are resolved by the realtime packages; these handlers retain shared
// mechanics, history and isolated practice-session endpoints.
package handlers
