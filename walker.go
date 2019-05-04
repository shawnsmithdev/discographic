package main

import (
	"encoding/hex"
	"github.com/shawnsmithdev/tag"
	"golang.org/x/sync/errgroup"
	"log"
	"os"
	"path/filepath"
	"time"
)

type walkResult struct {
	path    string
	size    int64
	modTime time.Time
}

func walker(out chan *walkResult) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			out <- &walkResult{
				path:    path,
				size:    info.Size(),
				modTime: info.ModTime().UTC(),
			}
		}
		return err
	}
}

func runWalker(root string, paths chan *walkResult) {
	defer close(paths)
	// TODO: Check that this is a folder first
	forbidErr(filepath.Walk(root, walker(paths)))
}

func readMeta(path string) (tag.Metadata, songHash, error) {
	var (
		songFile  *os.File
		meta      tag.Metadata
		tagHash   string
		hash      songHash
		foundHash = false
		err       error
	)
	if songFile, err = os.Open(path); err != nil {
		return nil, hash, err
	}
	if meta, err = tag.ReadFrom(songFile); err != nil {
		return nil, hash, nil // not an audio file, or otherwise unreadable
	}
	if _, err = songFile.Seek(0, 0); err != nil {
		return nil, hash, err
	}
	if meta.FileType() == tag.FLAC {
		if hash, err = flacMd5(songFile); err == nil {
			foundHash = true
		}
	}
	if !foundHash {
		if tagHash, err = tag.Sum(songFile); err != nil {
			return nil, hash, err
		} else if decoded, err := hex.DecodeString(tagHash); err != nil {
			return nil, hash, err
		} else {
			copy(hash[:], decoded)
		}
	}
	return meta, hash, songFile.Close()
}

type songAndArt struct {
	song *Song
	art  *Art
}

func handleSongWalk(wr *walkResult, out chan songAndArt) error {
	meta, hash, err := readMeta(wr.path)
	if err == nil {
		if meta == nil { // probably not a song
			return nil
		}
		hash64 := hash.String()
		file := hash64
		ext := filepath.Ext(wr.path)
		if len(ext) > 0 {
			file += ext
		}

		// song
		song := &Song{
			File:     file,
			MetaFile: hash64 + ".json",
			Size:     wr.size,
			ModTime:  wr.modTime,
			Path:     wr.path,
			Hash:     hash,
		}
		song.copyMetadata(meta)

		// art
		pic := meta.Picture()
		var songArt *Art
		if pic != nil {
			songArt = &Art{
				Data:     pic.Data,
				Ext:      pic.Ext,
				MimeType: pic.MIMEType,
			}
		}

		out <- songAndArt{
			song: song,
			art:  songArt,
		}
	} else {
		log.Println("meta error", err)
	}
	return err
}

func runSongWalkers(root string, parallel int) chan songAndArt {
	paths := make(chan *walkResult, parallel*16)
	go runWalker(root, paths)

	out := make(chan songAndArt, parallel*2)
	go func() {
		defer close(out)
		var eg errgroup.Group
		for i := 0; i < parallel; i++ {
			eg.Go(func() error {
				for result := range paths {
					if err := handleSongWalk(result, out); err != nil {
						return err
					}
				}
				return nil
			})
		}
		forbidErr(eg.Wait())
	}()
	return out
}

func runPathWalkers(mobile string) map[string]struct{} {
	paths := make(chan *walkResult, 256)
	go runWalker(mobile, paths)
	result := make(map[string]struct{})
	var nothing struct{}
	for wr := range paths {
		result[wr.path] = nothing
	}
	return result
}
