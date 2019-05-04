package main

import (
	"github.com/shawnsmithdev/tag"
	"strconv"
	"strings"
	"time"
)

const (
	// 20 byte sha1 from tag
	// or 16 byte md5 for flac with md5 tag + 4 byte flac suffix
	songHashSize = 20
)

type songHash [songHashSize]byte

func (s songHash) String() string {
	return bytesToString(s[:])
}

type Song struct {
	File        string       `json:"file"`                   // audio hash.ext
	MetaFile    string       `json:"meta_file"`              // audio hash.json
	Size        int64        `json:"size"`                   // size in bytes
	ModTime     time.Time    `json:"mod_time"`               // last modified time
	Album       string       `json:"album"`                  // ex. Muse
	Artist      string       `json:"artist"`                 // Used for compilations
	AlbumArtist string       `json:"album_artist,omitempty"` // ex. Origin of Symmetry
	Composer    string       `json:"composer,omitempty"`     // Used for classical music and compilations
	Title       string       `json:"title,omitempty"`        // ex. Space Dementia
	Track       int          `json:"track"`                  // Used to sort songs within albums
	Disc        int          `json:"disc,omitempty"`         // ex. Stadium Arcadium has 2
	Art         string       `json:"art,omitempty"`          // artwork hash.ext
	Comment     string       `json:"comment,omitempty"`      // freeform text
	FileType    tag.FileType `json:"file_type,omitempty"`    // ex. flac, mp3, m4a, ogg
	Date        string       `json:"date,omitempty"`         // one hopes this is ISO-8601, used to sort albums

	Path       string     `json:"-"` // filesystem path
	Hash       songHash   `json:"-"` // metadata agnostic audio hash
	MetaFormat tag.Format `json:"-"` // ex. vorbis, id3, mp4
}

const OPUS tag.FileType = "OPUS"

// Everything we care about from metadata is copied, except artwork
func (s *Song) copyMetadata(meta tag.Metadata) {
	s.Album = meta.Album()
	s.Artist = meta.Artist()
	s.AlbumArtist = meta.AlbumArtist()
	s.Composer = meta.Composer()
	s.Title = meta.Title()
	s.Track, _ = meta.Track()
	s.Disc, _ = meta.Disc()
	s.MetaFormat = meta.Format()

	// FileType... work around m4a detection bug
	s.FileType = meta.FileType()
	if s.FileType == tag.UnknownFileType {
		if s.MetaFormat == tag.MP4 && strings.HasSuffix(s.Path, ".m4a") {
			s.FileType = tag.M4A
		} else if s.MetaFormat == tag.VORBIS && strings.HasSuffix(s.Path, ".opus") {
			s.FileType = OPUS
		}
	}

	if len(meta.Date()) >= 4 {
		s.Date = meta.Date()
	} else if meta.Year() > 0 {
		s.Date = strconv.Itoa(meta.Year())
	}
}
