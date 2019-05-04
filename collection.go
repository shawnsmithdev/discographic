package main

import (
	"github.com/shawnsmithdev/tag"
)

// Collection is a recursively organized grouping of songs.
// TODO: hash
type Collection struct {
	// The human-friendly name of this collection, usually the artist or album name.
	Name string `json:"name"`
	// Collections that are contained by this one.
	Children []Collection `json:"children,omitempty"`
	// Songs that are contained in this collection.
	SongFiles []string `json:"song_files,omitempty"`
	// First song of all children, or empty if no children.
	FirstSong string `json:"first_song,omitempty"`
	// The library this collection describes.
	lib Library
}

// SongCount return the total count of songs in this collection, including child collections.
func (c Collection) SongCount() int {
	count := len(c.SongFiles)
	for _, child := range c.Children {
		count += child.SongCount()
	}
	return count
}

func (c Collection) TypeCount() map[tag.FileType]int {
	result := make(map[tag.FileType]int)
	c.Songs(func(song *Song) {
		ftype := song.FileType
		result[ftype] = result[ftype] + 1
	})
	for _, child := range c.Children {
		for ftype, count := range child.TypeCount() {
			result[ftype] = result[ftype] + count
		}
	}
	return result
}

func (c Collection) Songs(forEach func(song *Song)) {
	for _, songFile := range c.SongFiles {
		songHash, err := extractSongHash(songFile)
		forbidErr(err)
		if song := c.lib.findSong(songHash); song.File != "" {
			forEach(song)
		}
	}
}
