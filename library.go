package main

import (
	"crypto/sha512"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	megabyte = 1024 * 1024
	// SHA512_256 hash is 256 bits = 32 bytes
	picHashSize = 32
)

// picHash is a 32 byte array that represents the SHA512_256 hash of a picture file.
type picHash [picHashSize]byte

func (p picHash) String() string {
	return bytesToString(p[:])
}

type Art struct {
	Data     []byte
	Ext      string
	MimeType string
}

// Library is a set of audio files keyed by a metadata agnostic audio hash
type Library interface {
	artCount() int
	findArt(key picHash) *Art
	findSong(key songHash) *Song
	path() string
	putSongAndArt(songAndArt songAndArt, logger *log.Logger)
	songCount() int
	songs(toDo func(*Song) error) error
	storeDb(db string) error
}

type library struct {
	SongMap map[songHash]*Song
	ArtMap  map[picHash]*Art
}

func (l library) artCount() int {
	return len(l.ArtMap)
}

func (l library) findArt(key picHash) *Art {
	return l.ArtMap[key]
}

func (l library) findSong(key songHash) *Song {
	return l.SongMap[key]
}

func (l library) path() string {
	panic("implement me")
}

func (l *library) putSongAndArt(songAndArt songAndArt, logger *log.Logger) {
	song := songAndArt.song
	songArt := songAndArt.art
	if songArt == nil {
		logger.Printf("No art found for song at %q", songAndArt.song.Path)
		song.Art = ""
	} else {
		hash := picHash(sha512.Sum512_256(songArt.Data))
		if _, ok := l.ArtMap[hash]; !ok {
			l.ArtMap[hash] = songArt
		}
		artFile := hash.String()
		artExt := songArt.Ext
		if len(artExt) > 0 {
			artFile = fmt.Sprintf("%s.%s", artFile, artExt)
		}
		song.Art = artFile
	}
	l.SongMap[songAndArt.song.Hash] = song
}

func (l library) songCount() int {
	return len(l.SongMap)
}

func (l library) songs(forEach func(*Song) error) error {
	for _, song := range l.SongMap {
		if err := forEach(song); err != nil {
			return err
		}
	}
	return nil
}

func newLibrary() *library {
	return &library{
		SongMap: make(map[songHash]*Song),
		ArtMap:  make(map[picHash]*Art),
	}
}

func loadDb(db string) *library {
	f, err := os.Open(db)
	forbidErr(err)
	defer closeFile(f)
	result := &library{}
	forbidErr(gob.NewDecoder(f).Decode(result))
	return result
}

func (l library) storeDb(db string) error {
	f, err := os.Create(db)
	if err != nil {
		return err
	}
	err = gob.NewEncoder(f).Encode(&l)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	return f.Close()
}

type loadLibraryArgs struct {
	root     string
	parallel int
	logger   *log.Logger
	db       string
	rescan   bool
}

func loadLibrary(args loadLibraryArgs) Library {
	var result Library
	if "" != args.db && !args.rescan {
		args.logger.Printf("will load library from db at %q", args.db)
		start := time.Now()
		result = loadDb(args.db)
		delta := time.Now().Sub(start)
		args.logger.Printf("loaded library from db at %q with %v songs and %v pics in %v",
			args.db, result.songCount(), result.artCount(), delta)
		return result
	}
	start := time.Now()
	result = newLibrary()
	total := int64(0)
	for songAndArt := range runSongWalkers(args.root, args.parallel) {
		args.logger.Printf("found song, path=%q", songAndArt.song.Path)
		result.putSongAndArt(songAndArt, args.logger)
		total += songAndArt.song.Size
	}
	delta := time.Now().Sub(start)
	speed := float64(total) / (delta.Seconds() * megabyte)
	args.logger.Printf("loaded library with %v songs and %v pics in %v (%.0f MB/s)",
		result.songCount(), result.artCount(), delta, speed)
	if "" != args.db {
		args.logger.Printf("will store library to db at %q", args.db)
		start = time.Now()
		if err := result.storeDb(args.db); err == nil {
			delta := time.Now().Sub(start)
			args.logger.Printf("stored library to db at %q with %v songs and %v pics in %v",
				args.db, result.songCount(), result.artCount(), delta)
		} else {
			args.logger.Printf("failed to store library to db at %q", args.db)
			args.logger.Print(err)
		}
	}
	return result
}

func closeFile(f *os.File) {
	forbidErr(f.Close())
}
