package main

import (
	"encoding/base64"
	"flag"
	"log"
	"os"
	"strings"
)

const (
	uiAddress         = ":61337"
	contentTypeHeader = "Content-Type"
	jsonMime          = "application/json"
	mp3Mime           = "audio/mpeg"
	m4aMime           = "audio/mp4"
	flacMime          = "audio/flac"
	backupMime        = "application/octet-stream"
	minParallel       = 1
	maxParallel       = 64
)

func main() {
	var (
		root       string
		parallel   int
		gui        bool
		address    string
		db         string
		doRescanDb bool

		mobile       string
		doSyncMobile bool
	)
	flag.StringVar(&root, "root", "", "root music library folder")
	flag.IntVar(&parallel, "p", 1, "parallelism of library loading")
	flag.BoolVar(&gui, "gui", true, "enable web gui")
	flag.StringVar(&address, "address", uiAddress, "address to listen to for rest api and gui")
	flag.StringVar(&db, "database", "", "location of database file, default is no persistence")
	flag.BoolVar(&doRescanDb, "rescan-database", false, "if true, rescans existing database")
	flag.StringVar(&mobile, "mobile", "", "optional mobile music library folder")
	flag.BoolVar(&doSyncMobile, "sync-mobile", false, "run mobile library sync")

	flag.Parse()
	if len(root) == 0 {
		panic("Must provide --root argument for music library root folder")
	}
	if parallel < minParallel {
		parallel = minParallel
	} else if parallel > maxParallel {
		parallel = maxParallel
	}

	loadLog := log.New(os.Stdout, "[load] ", log.LstdFlags|log.Lmicroseconds)
	lib := loadLibrary(loadLibraryArgs{
		root:     root,
		parallel: parallel,
		logger:   loadLog,
		db:       db,
		rescan:   doRescanDb,
	})
	aad := ArtistAblumDateCollection(lib, loadLog)

	loadLog.Println("================================")
	if "" != mobile {
		loadLog.Println("mobile library:", mobile)
		if doSyncMobile {
			loadLog.Print(mobileSyncWarning)
			forbidErr(syncMobile(ensurePathSep(root), lib, ensurePathSep(mobile)))
			return
		}
	}
	loadLog.Println("================================")
	server := buildServer(aad, lib, gui)
	server.Addr = address

	logAddress := address
	if strings.HasPrefix(logAddress, ":") {
		logAddress = "localhost" + logAddress
	}
	loadLog.Printf("Serving UI on http://%v\n", logAddress)
	forbidErr(server.ListenAndServe())
}

// Don't use this on user error, use 4XX HTTP codes instead.
// Don't use this for ordinary or exceptional errors, use errors and 5xx HTTP codes instead.
// This is fine for things that should be absolutely impossible (server bugs).
func forbidErr(err error) {
	if err != nil {
		log.Println("forbidden error!", err)
		panic(err)
	}
}

// converts a byte slice into an url safe base64 string without padding
func bytesToString(data []byte) string {
	hash64 := base64.URLEncoding.EncodeToString(data)
	return strings.Replace(hash64, "=", "", -1)
}
