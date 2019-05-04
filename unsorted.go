package main

import "strings"

type artistAlbum map[string]unsorted

type unsorted map[string][]*Song

func (aa artistAlbum) addAll(lib Library) error {
	return lib.songs(func(song *Song) error {
		var artistKey string
		if len(song.AlbumArtist) > 0 {
			artistKey = strings.ToLower(song.AlbumArtist)
		} else {
			artistKey = strings.ToLower(song.Artist)
		}
		sameArtist, ok := aa[artistKey]
		if !ok {
			sameArtist = make(unsorted)
		}
		albumKey := strings.ToLower(song.Album)
		sameAlbum := sameArtist[albumKey]
		sameArtist[albumKey] = append(sameAlbum, song)
		aa[artistKey] = sameArtist
		return nil
	})
}
