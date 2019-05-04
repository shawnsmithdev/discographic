package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dimfeld/httptreemux/v5"
	"github.com/shawnsmithdev/tag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func buildServer(aad Collection, lib Library, gui bool) *http.Server {
	router := httptreemux.NewContextMux()
	router.PanicHandler = httptreemux.ShowErrorsPanicHandler
	router.PathSource = httptreemux.URLPath

	if gui {
		router.GET("/", http.FileServer(http.Dir("./static/")).ServeHTTP)
		for _, prefix := range []string{"js", "css", "templates"} {
			path := fmt.Sprintf("/%s/*", prefix)
			strip := fmt.Sprintf("/%s", prefix)
			dir := fmt.Sprintf("./static/%s", prefix)
			handler := http.FileServer(http.Dir(dir))
			router.GET(path, http.StripPrefix(strip, handler).ServeHTTP)
		}
	}

	restLog := log.New(os.Stdout, "[rest] ", log.LstdFlags|log.Lmicroseconds)
	router.GET("/music/aad.json", aadHandler(aad, restLog))
	router.GET("/music/metadata/:song", metaHandler(lib, restLog))
	router.GET("/music/song/:song", songHandler(lib, restLog))
	router.GET("/music/raw/:song", rawHandler(lib))
	router.GET("/music/art/:art", artHandler(lib, restLog))

	return &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      1 * time.Minute,
		ErrorLog:          restLog,
		IdleTimeout:       1 * time.Hour,
	}
}

func writeNotFoundErr(w http.ResponseWriter, logger *log.Logger, err error) {
	logger.Println(err)
	http.Error(w, fmt.Sprint(err), http.StatusNotFound)
}

func writeYourErr(w http.ResponseWriter, logger *log.Logger, err error) {
	logger.Println(err)
	http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
}

func songHandler(lib Library, logger *log.Logger) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		if songArg, ok := httptreemux.ContextParams(req.Context())["song"]; ok {
			if songHash, err := extractSongHash(songArg); err == nil {
				if song := lib.findSong(songHash); song.File != "" {
					switch req.Method {
					case "GET":
						logger.Printf("serving song=%v, path=%q", songArg, song.Path)
					case "HEAD":
						logger.Printf("head on song=%v, path=%q", songArg, song.Path)
					}
					switch song.FileType {
					case tag.ALAC:
						fallthrough
					case tag.M4A:
						writer.Header().Set(contentTypeHeader, m4aMime)
					case tag.MP3:
						writer.Header().Set(contentTypeHeader, mp3Mime)
					case tag.FLAC:
						writer.Header().Set(contentTypeHeader, flacMime)
					default:
						writer.Header().Set(contentTypeHeader, backupMime)
					}
					http.ServeFile(writer, req, song.Path)
					return
				}
			}
			writeNotFoundErr(writer, logger, fmt.Errorf("couldn't find song: %v", songArg))
		} else {
			writeYourErr(writer, logger, fmt.Errorf("path requires 1 argument (song hash)"))
		}
	}
}

func extractPicHash(file string) (picHash, error) {
	var result picHash
	hash, err := extractHash(file)
	if err == nil {
		copy(result[:], hash)
	}
	return result, err
}

func extractSongHash(file string) (songHash, error) {
	var result songHash
	hash, err := extractHash(file)
	if err == nil {
		copy(result[:], hash)
	}
	return result, err
}

func extractHash(file string) ([]byte, error) {
	asBase64 := file
	ext := filepath.Ext(file)
	if len(ext) > 0 {
		asBase64 = file[:len(file)-len(ext)]
	}
	return base64.URLEncoding.DecodeString(asBase64 + "=")
}

func artHandler(lib Library, logger *log.Logger) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		if artArg, ok := httptreemux.ContextParams(req.Context())["art"]; ok {
			if artHash, err := extractPicHash(artArg); err == nil {
				if art := lib.findArt(artHash); art != nil {
					logger.Printf("serving art %v", artArg)
					writer.Header().Set(contentTypeHeader, art.MimeType)
					_, err := writer.Write(art.Data)
					forbidErr(err)
					return
				}
			}
			writeNotFoundErr(writer, logger, fmt.Errorf("unknown art file: %v", artArg))
		} else {
			writeYourErr(writer, logger, fmt.Errorf("path requires 1 argument (art hash)"))
		}
	}
}

func metaHandler(lib Library, logger *log.Logger) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		if songArg, ok := httptreemux.ContextParams(req.Context())["song"]; ok {
			if songHash, err := extractSongHash(songArg); err == nil {
				if song := lib.findSong(songHash); song.File != "" {
					logger.Printf("serving song metadata for %v", songArg)
					result, err := json.Marshal(song)
					forbidErr(err)
					writer.Header().Set(contentTypeHeader, jsonMime)
					_, err = writer.Write(result)
					forbidErr(err)
					return
				}
			}
			writeNotFoundErr(writer, logger, fmt.Errorf("couldn't find song for hash: %v", songArg))
		} else {
			writeYourErr(writer, logger, fmt.Errorf("path requires 1 argument (song hash)"))
		}
	}
}

// Handler for artist-album-date collection
func aadHandler(col Collection, logger *log.Logger) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		logger.Println("serving aad collection, song_count:", col.SongCount())
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(col)
		forbidErr(err)
		writer.Header().Set(contentTypeHeader, jsonMime)
		_, err = writer.Write(buf.Bytes())
		forbidErr(err)
	}
}

// TODO: Add secret toggle in gui to expose this data for use while debugging tag package
func rawHandler(lib Library) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set(contentTypeHeader, jsonMime)
		if songArg, ok := httptreemux.ContextParams(req.Context())["song"]; ok {
			if songHash, err := extractSongHash(songArg); err == nil {
				if song := lib.findSong(songHash); song.Path != "" {
					meta, _, err := readMeta(song.Path)
					forbidErr(err)
					if len(meta.Raw()) > 0 {
						buf := new(bytes.Buffer)
						err = json.NewEncoder(buf).Encode(meta.Raw())
						forbidErr(err)
						_, err = writer.Write(buf.Bytes())
						forbidErr(err)
						return
					}
				}
			}
		}
		_, err := writer.Write([]byte("{}"))
		forbidErr(err)
	}
}
