package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlushBatchIntoDB(t *testing.T) {

	db, err := InitDB()
	require.NoError(t, err)

	batch := []*Comic{
		&Comic{
			Num:   1,
			Alt:   "bla",
			Img:   "http://ballz.com",
			Title: "ballz",
		},
		&Comic{
			Num:   2,
			Alt:   "bla",
			Img:   "http://ballz.com",
			Title: "ballz",
		},
		&Comic{
			Num:   3,
			Alt:   "bla",
			Img:   "http://ballz.com",
			Title: "ballz",
		},
	}
	
	err = db.FlushBatch(batch)
	require.NoError(t, err)
}

func TestQuery(t *testing.T){	
	db, err := InitDB()
	require.NoError(t, err)

	keywords := []string{"comic"}
	err = db.QueryComics(keywords...)
	require.NoError(t, err)
}