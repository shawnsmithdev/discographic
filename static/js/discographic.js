new Vue({
    el: "#discographic",
    data: {
        collection: [],
        browseList: [],
        breadcrumb: [{name: "Home", idx: -1}],
        loadedCollection: false,
        loadedPlaylist: false,
        playQueue: [],
        playQueueShow: -1,
        currentSong: -1,
        player: null
    },
    mounted: function () {
        if (this.$refs.hasOwnProperty("mpaudio")) {
            this.player = this.$refs.mpaudio;
            this.player.addEventListener("ended", () => this.changeSong(1));
        } else {
            console.log("did not find player, no event listeners added!");
        }
    },
    methods: {
        changeSong: function (delta) {
            console.log("changing song, delta:" + delta);
            let deltaIdx = this.currentSong + delta;
            if (typeof this.playQueue[deltaIdx] === "undefined") {
                console.log("failed changing song:" + deltaIdx);
                return false;
            }
            this.currentSong = deltaIdx;
            this.player.pause();
            this.player.setAttribute("src", "/music/song/" + this.playQueue[this.currentSong].file);
            this.player.load();
            this.player.play();
            this.showQueueInfo(deltaIdx);
        },
        loadQueueItem: function (idx) {
            let freshQueue = [];
            if (this.breadcrumb.length === 1) {
                let discography = this.collection[idx];
                for (let album of discography.children) {
                    for (let metaFile of album.song_files) {
                        freshQueue.push({meta_file: metaFile, title: "..."});
                    }
                }
                freshQueue[0].file = discography.first_song;
            } else if (this.breadcrumb.length === 2) {
                let album = this.collection[this.breadcrumb[1].idx].children[idx];
                for (let metaFile of album.song_files) {
                    freshQueue.push({meta_file: metaFile, title: "..."});
                }
                freshQueue[0].file = album.first_song;
            }
            this.playQueue = freshQueue;
            this.playQueueShow = -1;
            this.currentSong = 0;
            this.changeSong(0);
            this.loadedPlaylist = true;
            this.updatePlayQueue();
        },
        browseUp: function (idx) {
            this.breadcrumb = this.breadcrumb.slice(0, idx+1);
            if (idx === 0) { // entire collection
                this.browseList = this.collection;
            } else if (idx === 1) { // artist discography
                this.browseList = this.collection[this.breadcrumb[1].idx].children;
            }
        },
        browseDown: function (idx) {
            if (this.breadcrumb.length === 1) { // artist discography
                let discography = this.collection[idx];
                this.breadcrumb.push({name: discography.name, idx: idx});
                this.browseList = discography.children;

            } else if (this.breadcrumb.length === 2) { // album
                let album = this.collection[this.breadcrumb[1].idx].children[idx];
                this.breadcrumb.push({name: album.name, idx: idx});

                // song placeholders
                let metaFiles = album.song_files;
                let songs = [];
                for (let i = 0; i < metaFiles.length; i++) {
                    songs.push({meta_file: metaFiles[i], title: "..."});
                }
                this.browseList = songs;

                // lookup metadata to replace placeholders
                this.updateBrowselist();
            }
        },
        updateMetadata: function (metaFile, thenFunc) {
            axios.get("/music/metadata/" + metaFile)
                .then(response => {
                    thenFunc(response.data);
                })
                .catch(error => {
                    console.log(error);
                });
        },
        updatePlayQueue: function () {
            for (let i = 0; i < this.playQueue.length; i++) {
                this.updateMetadata(this.playQueue[i].meta_file, meta => {
                    if (meta.hasOwnProperty("size")) {
                        meta["show_size"] = filesize(meta.size) + " (" + meta.size + ")";
                    }
                    Vue.set(this.playQueue, i, meta);

                    // last modified is from head of song file TODO: put in metadata
                    axios.head("/music/song/" + meta.file).then(response => {
                        if (response.headers.hasOwnProperty("last-modified")) {
                            Vue.set(this.playQueue[i], "last_modified", response.headers["last-modified"]);
                        }
                    })
                });
            }
        },
        updateBrowselist: function () {
            for (let i = 0; i < this.browseList.length; i++) {
                this.updateMetadata(this.browseList[i].meta_file, meta => {
                    Vue.set(this.browseList, i, meta);
                });
            }
        },
        loadCollection: function () {
            axios.get("/music/aad.json")
                .then(response => {
                    this.collection = response.data.children;
                    this.loadedCollection = this.collection.length > 0;
                    this.browseList = this.collection;
                })
                .catch(error => {
                    console.log(error);
                });
        },
        showQueueInfo: function (idx) {
            if (this.playQueueShow === idx) {
                this.playQueueShow = -1;
            } else {
                this.playQueueShow = idx;
            }
        }
    }
});
