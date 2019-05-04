package main

import (
	"bytes"
	"github.com/shawnsmithdev/tag"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const mobileSyncWarning = `Only audio, jpg/jpeg, and png files are synced to the mobile library.
ALL EXISTING FILES in the mobile library that are not audio, jpg, or png WILL BE ERASED.
ALL EXISTING FILES in the mobile library that do not have a related file in the root library WILL BE ERASED.
ALL EXISTING FILES in the mobile library that are older than the related file in the root library WILL BE OVERWRITTEN.
`

// syncMobile overwrites the contents of the filesystem at mobile with the audio and picture files
// of the filesystem at root, except where a file is a FLAC audio file, where instead
// an opus encoded copy may be made.
// If mobile does not exist yet, it will be created.
// Files are not overwritten if already present with the same or newer modified timestamp as root.
func syncMobile(root string, lib Library, mobile string) error {
	// pre-fill audio files we already know we want
	rootPaths := make(map[string]string)
	forbidErr(lib.songs(func(song *Song) error {
		rootPath := song.Path[len(root):]
		rootPaths[rootPath] = string(song.FileType)
		return nil
	}))
	// add any art files
	allRoot := make(chan *walkResult)
	go runWalker(root, allRoot)
	for wr := range allRoot {
		if _, ok := rootPaths[wr.path]; ok {
			continue
		}
		rootPath := wr.path[len(root):]
		ext := strings.ToLower(filepath.Ext(wr.path))
		switch ext {
		case ".jpeg":
			fallthrough
		case ".jpg":
			rootPaths[rootPath] = "JPG"
		case ".png":
			rootPaths[rootPath] = "PNG"
		} // ignore anything else
	}

	// get existing mobile files
	mobilePaths := runPathWalkers(mobile)

	// delete unknown
	// TODO: mod time check
	// TODO: Consolidate with empty folders delete into one pass
	log.Println("deleting unknown files")
	var deleted []string
	for path := range mobilePaths {
		mobilePath := path[len(mobile):]
		if _, ok := rootPaths[mobilePath]; ok {
			continue
		}
		mobileExt := filepath.Ext(mobilePath)
		if mobileExt == ".opus" {
			mobileBase := mobilePath[:len(mobilePath)-len(mobileExt)]
			flacPath := mobileBase + ".flac"
			if _, ok := rootPaths[flacPath]; ok {
				continue
			}
		}
		if err := os.Remove(path); err == nil {
			log.Printf("deleted: %q\n", path)
			deleted = append(deleted, mobilePath)
		} else {
			log.Println(err)
		}
	}
	for _, d := range deleted {
		delete(mobilePaths, d)
	}

	// delete empty folders
	log.Println("deleting empty folders")
	err := filepath.Walk(mobile, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() || mobile == path {
			return nil
		}
		if file, err := os.Open(path); err == nil {
			if _, err := file.Readdir(1); io.EOF != err {
				return err
			}
			err := os.Remove(path)
			if err == nil {
				log.Println("removed empty directory", path)
			}
			return err
		} else {
			return err
		}
	})

	if err != nil {
		return err
	}

	// copy lossy and art
	toEncode := make(chan encodeTask, 64)
	encodeErr := make(chan error)
	go func() {
		encodeErr <- encode(toEncode)
		log.Println("encoding complete")
		close(encodeErr)
	}()
	for path, fileType := range rootPaths {
		if fileType == string(tag.FLAC) {
			rootExt := filepath.Ext(path)
			rootBase := path[:len(path)-len(rootExt)]
			mobilePath := rootBase + ".opus"
			if _, ok := mobilePaths[mobilePath]; ok {
				log.Println("Already exists:", mobile+mobilePath)
			} else {
				toEncode <- encodeTask{
					inPath:  root + path,
					outPath: mobile + mobilePath,
				}
			}
			continue
		}
		if _, ok := mobilePaths[path]; ok {
			log.Println("Already exists:", mobile+path)
			continue
		}
		if err := copyFile(root+path, mobile+path); err != nil {
			close(toEncode)
			return err
		}
	}
	close(toEncode)
	err = <-encodeErr
	if err != nil {
		log.Println(err)
	}
	return err
}

func ensureFolders(outFilePath string) error {
	return os.MkdirAll(filepath.Dir(outFilePath), os.ModePerm)
}

func writeFile(data []byte, outFilePath string) error {
	start := time.Now()
	if err := ensureFolders(outFilePath); err != nil {
		return err
	}
	if outFile, err := os.Create(outFilePath); err == nil {
		defer func() {
			forbidErr(outFile.Sync())
			forbidErr(outFile.Close())
		}()
		outCount, err := outFile.Write(data)
		if err == nil {
			log.Printf("Write new file (size %v) to %q in %v\n",
				outCount, outFilePath, time.Now().Sub(start))
		}
		return err
	} else {
		return err
	}
}

func copyFile(inFilePath, outFilePath string) error {
	start := time.Now()
	if err := ensureFolders(outFilePath); err != nil {
		return err
	}
	if outFile, err := os.Create(outFilePath); err == nil {
		defer func() {
			forbidErr(outFile.Sync())
			forbidErr(outFile.Close())
		}()
		if inFile, err := os.Open(inFilePath); err == nil {
			outCount, copyErr := io.Copy(outFile, inFile)
			if copyErr == nil {
				log.Printf("Copied file (size %v) from %q to %q in %v\n",
					outCount, inFilePath, outFilePath, time.Now().Sub(start))
			}
			return copyErr
		} else {
			return err
		}
	} else {
		return err
	}
}

func ensurePathSep(path string) string {
	if len(path) > 0 && !os.IsPathSeparator(path[len(path)-1]) {
		return path + string(os.PathSeparator)
	}
	return path
}

// Convert a FLAC file to an byte array representing an opus file
// This requires "opusenc" to be available in the execution path
// Ensure that this path is for known good flac file.
func flacToOpus(path string) ([]byte, error) {
	start := time.Now()
	opusenc := exec.Command("opusenc", "--bitrate", "256", path, "-")
	var out bytes.Buffer
	opusenc.Stdout = &out
	if err := opusenc.Run(); err != nil {
		return nil, err
	} else {
		log.Printf("Encode opus (size %d) from %q in %v", out.Len(), path, time.Now().Sub(start))
		return out.Bytes(), nil
	}
}

type encodeTask struct {
	inPath  string
	outPath string
}

func encode(tasks chan encodeTask) error {
	parallel := runtime.NumCPU()
	var eg errgroup.Group
	for i := 0; i < parallel; i++ {
		eg.Go(func() error {
			for task := range tasks {
				data, err := flacToOpus(task.inPath)
				if err != nil {
					return err
				}
				err = writeFile(data, task.outPath)
				if err != nil {
					log.Println("writeFile err", err)
					return err
				}
			}
			return nil
		})
	}
	return eg.Wait()
}
