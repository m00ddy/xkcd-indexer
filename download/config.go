package download

import (
	"github.com/ripp4rd0c/xkcd/db"
	"github.com/ripp4rd0c/xkcd/logger"
)

const (
	Table = "comics"
)

type Config struct{
	Logger *logger.Wood
	Db db.ComicsDB
	MaxComics int
	URL string
}

