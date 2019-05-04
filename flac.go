package main

import (
	"fmt"
	"os"
)

const (
	// First 4 bytes is FLAC file header, used here as a suffix for FLAC hashes.
	// This gives them the same length as SHA1 hashes.
	flacHeader = "fLaC"
	// MD5 hash starts after:
	// flac header (4)
	// STREAMINFO header (4)
	// other STREAMINFO data (18)
	hashStart = 26
	// MD5 hashes are 128 bits = 16 bytes
	md5Bytes = 16
)

func flacMd5(songFile *os.File) (songHash, error) {
	// best effort file reset
	defer func() {
		_, _ = songFile.Seek(0, 0)
	}()

	var buffer songHash

	// while we are here, confirm this really is a flac file
	if _, err := songFile.ReadAt(buffer[:len(flacHeader)], 0); err != nil {
		return buffer, err
	} else if flacHeader != string(buffer[:len(flacHeader)]) {
		return buffer, fmt.Errorf("not a flac file")
	}

	// read MD5
	if _, err := songFile.ReadAt(buffer[:md5Bytes], hashStart); err != nil {
		return buffer, err
	}

	// check for all zero
	for _, b := range buffer[:md5Bytes] {
		if b != 0 { // not all zero, copy flac suffix
			copy(buffer[md5Bytes:], flacHeader)
			return buffer, nil
		}
	}

	// all zero (not set)
	return buffer, fmt.Errorf("FLAC file %q has no MD5 checksum in its STREAMINFO block\n", songFile.Name())
}
