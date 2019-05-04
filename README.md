[![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/shawnsmithdev/discographic/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/shawnsmithdev/discographic)](https://goreportcard.com/report/github.com/shawnsmithdev/discographic)

About
=====

`discographic` is a server daemon that scans a directory of music and serves music files and metadata over a REST api.
Browsing structure is defined using metadata read from file tags, but actual music resources are referenced by metadata
agnostic hashes. This allows updates to metadata tags by other applications to only affect browsing, leaving playback of
existing playlists and queues unaffected.

`discographic` can be compared to mpd (Music Player Daemon), except that `discographic` does not play back music itself,
as the client is expected to decode and play audio files directly. The REST api only serves the music files and browsing
metadata.

Another similar tool is Plex, however Plex has video support, whereas `discographic` only supports music collections.

The tool comes bundled with an extremely basic web app client built using vue.js and html5 audio tags.
The long term goal is to support native apps with the flexibilty of foobar2000 but with a REST-based server-client
design, along with much more advanced web apps that would likely use the
[Web Audio API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Audio_API/Basic_concepts_behind_Web_Audio_API).

Usage
=====
``` bash
# Complete usage for all flags.
./discographic --help

# Music root folder path is required.
# Load paralellism of 32 makes loads faster on ssd, but will make loads slower on a mechanical hard drive.
./discographic -root ~/Music -p 32

# Synchronize the root music folder with another folder that stores a mobile library
# This transcodes all FLAC files in the root library to Opus
# This will clobber files in the mobile library, whereas root libraries are always read only.
./discographic -root ~/Music -mobile ~/PhoneMusic -sync-mobile

# Store the results of scanning the root library in a database file
./discographic -root ~/Music -database ~/disco.db -rescan-database

# Use previously stored database to quickly start daemon without scanning the library again
./discographic -root ~/Music -database ~/disco.db
```

Implemented Features
====================
* Scan music, presents REST api for supported file types (FLAC, AAC/MP4, MP3, OGG)
* Extremely basic web UI
* Basic library persistence using gob-based database file
* Optional secondary library for small devices (for ex. cell phones, keeps lossy, encodes flac to opus)

Planned Features
================
[ ] Opus support in main music library (needs fix for tag library)
[ ] Fix web ui play buttons
[ ] Flexible metadata queries using custom dsl (like foobar2000 has)
[ ] Organize UI by query results or file system structure, remove AlbumArtistDate api
[ ] Monitor file system changes, realtime library updates
[ ] Manually trigger root and subfolder rescans from api and ui.
[ ] Store incremental metadata changes in database file (avoid full rescan of music library on small changes)
[ ] Optional flac to opus transcoding during playback (low bandwidth, ex. home vpn)
[ ] Support reference of collections by hash of child or song hashes (Merkle Tree)
[ ] Support low max depth of metadata query results (requires collection references)
[ ] Better Web UI

Feature Graveyard
=================
* ALAC to OPUS support - Users must convert to ALAC to FLAC first
