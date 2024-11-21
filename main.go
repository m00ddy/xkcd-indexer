package main

// go run main.go -f/-q [(query) keywords...]

import (
	"flag"
	"fmt"
	"os"

	"github.com/ripp4rd0c/xkcd/db"
	"github.com/ripp4rd0c/xkcd/download"
	"github.com/ripp4rd0c/xkcd/logger"
)

var (
	// flags:
	fetch = flag.Bool("f", false, "update local comics repository")
	query = flag.Bool("q", false, "use keywords to search comics database")
)

func main() {
	logfile, err := os.OpenFile("download.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("error opening a log file")
	}
	defer func() { _ = logfile.Close() }()
	lg, err := logger.InitLogger(logfile)
	if err != nil {
		fmt.Println("error opening log file")
	}
	cdb, err := db.InitDB()
	if err != nil {
		lg.LogFatal("failed opening a database connection")
	}
	defer func() { _ = cdb.Close() }()

	c := download.Config{
		Logger: lg,
		Db:     cdb,
		URL:    "https://xkcd.com/%d/info.0.json",
	}

	flag.Parse()
	keywords := flag.Args()

	switch {

	case *fetch:
		fmt.Println("updating local comics repo")

		remoteLatest, err := download.LatestXkcd()
		if err != nil {
			lg.LogFatal("probing for latest xkcd failed, check internet")
		}
		localLatest, err := cdb.LastComic()
		if err != nil {
			lg.LogError(err)
			lg.LogFatal("can't get last local comic number")
		}

		// local repo needs updating
		if remoteLatest > localLatest {
			c.MaxComics = remoteLatest - localLatest
			offset := localLatest
			// start from the last comic number we have
			f, err := download.NewFetcher(&c)
			if err != nil {
				lg.LogError("fetcher initialization failed")
			}
			f.Download(offset)
		}

		fmt.Println("local comics repo up to date!")

	case *query:
		fmt.Println(keywords)
		if err := cdb.QueryComics(keywords...); err != nil {
			lg.LogFatal("error querying comics database")
		}
	}
}
