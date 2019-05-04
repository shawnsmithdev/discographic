package main

import (
	"log"
	"sort"
	"time"
)

// TODO: Support more sorting and grouping options by implementing a query dsl like foobar2000
// TODO: Support max depth
// ArtistAlbumDateCollection builds a collection by artist/album, sorting albums by date.
func ArtistAblumDateCollection(lib Library, logger *log.Logger) Collection {
	start := time.Now()
	aa := make(artistAlbum)
	forbidErr(aa.addAll(lib))

	sortedArtists, sortedAlbums := sortArtistAlbumDate(aa)

	result := Collection{
		Name: "ArtistAlbumDateCollection",
		lib:  lib,
	}
	for _, artist := range sortedArtists {
		discography := Collection{}
		for _, album := range sortedAlbums[artist] {
			albumSongs := aa[artist][album]
			var metas []string
			for _, song := range albumSongs {
				metas = append(metas, song.MetaFile)
			}
			hash, err := extractSongHash(metas[0])
			forbidErr(err)
			first := lib.findSong(hash)
			discography.Children = append(discography.Children, Collection{
				Name:      first.Album,
				SongFiles: metas,
				lib:       lib,
				FirstSong: first.File,
			})
		}
		discography.FirstSong = discography.Children[0].SongFiles[0]
		hash, err := extractSongHash(discography.FirstSong)
		forbidErr(err)
		first := lib.findSong(hash)
		discography.Name = first.AlbumArtist
		if len(discography.Name) == 0 {
			discography.Name = first.Artist
		}
		result.Children = append(result.Children, discography)
	}

	logger.Printf("organized Library into ArtistAlbumDate collection in %v", time.Now().Sub(start))
	logger.Printf("  Song Count:   %v", result.SongCount())
	logger.Printf("  Type Count:   %+v", result.TypeCount())
	return result
}

func sortArtistAlbumDate(aa artistAlbum) ([]string, map[string][]string) {
	var (
		artists []string
		albums  = make(map[string][]string)
	)

	for artistKey, discography := range aa {
		byDate := make(map[string][]string)
		for albumKey, songs := range discography {
			if len(songs) == 0 {
				continue
			}
			sort.Slice(songs, compareSongTrack(songs))
			date := songs[0].Date
			byDate[date] = append(byDate[date], albumKey)
		}

		var dates []string
		for date, dateAlbums := range byDate {
			dates = append(dates, date)
			sort.Strings(dateAlbums)
		}
		sort.Strings(dates)

		var sortedAlbums []string
		for _, date := range dates {
			for _, album := range byDate[date] {
				sortedAlbums = append(sortedAlbums, album)
			}
		}

		artists = append(artists, artistKey)
		albums[artistKey] = sortedAlbums
	}

	sort.Strings(artists)
	return artists, albums
}

func compareSongTrack(songs []*Song) func(i, j int) bool {
	return func(i, j int) bool {
		songI := songs[i]
		songJ := songs[j]
		discI := songI.Disc
		discJ := songJ.Disc
		if discJ != discI {
			return discI < discJ
		}
		trackI := songI.Track
		trackJ := songJ.Track
		if trackI > 0 && trackJ > 0 {
			return trackI < trackJ
		}
		// 0's shouldn't happen. Hope the file paths will be in better shape
		return songs[i].Path < songs[j].Path
	}
}
