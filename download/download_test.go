package download

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/ripp4rd0c/xkcd/db"
	"github.com/ripp4rd0c/xkcd/db/mocks"
	"github.com/ripp4rd0c/xkcd/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFetchComic(t *testing.T) {

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/info.0.json" {
			c := db.Comic{
				Num:   1,
				Title: "Test Comic",
				Img:   "https://example.com/test-comic.png",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(c)
		} else {
			// 404
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	comicPath := testServer.URL + "/1/info.0.json"
	missingComicPath := testServer.URL + "/404/info.0.json"

	c, err := fetchComic(comicPath, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if c.Num != 1 {
		t.Fatalf("expected comic NUM == 1, got %v", c.Num)
	}

	// simulate the 404 error
	_, err = fetchComic(missingComicPath, 404)
	// we expect an error
	if err == nil {
		t.Fatalf("expected error got nil")
	}
	// the error must be:
	expectedErr := fmt.Errorf("missing comic number %d", 404)
	if err.Error() != expectedErr.Error() {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestFetchBatch(t *testing.T) {
	//! the fetchBatch function is the thing being tested here

	/* initialization */
	//! create a test server that serves the comics we want to flush
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {	
		comics := []*db.Comic{
			{
				Num:   1,
				Alt:   "bla",
				Img:   "http://ballz.com",
				Title: "ballz",
			},
			{
				Num:   2,
				Alt:   "bla",
				Img:   "http://ballz.com",
				Title: "ballz",
			},
		}
		index := r.URL.Path
		re, _ := regexp.Compile(`(\d+)`)
		if !re.MatchString(index) {
			w.WriteHeader(http.StatusNotFound)	
		}else{
			i, _ := strconv.Atoi(re.FindString(index))
			fmt.Println("successful hit, serving comic number", i)
			json.NewEncoder(w).Encode(comics[i-1])
		}

	}))
	defer ts.Close()

	// prepend test server URL to the comic number
	url := ts.URL + "/%d/"

	expectedComics := []*db.Comic{
			{
				Num:   1,
				Alt:   "bla",
				Img:   "http://ballz.com",
				Title: "ballz",
			},
			{
				Num:   2,
				Alt:   "bla",
				Img:   "http://ballz.com",
				Title: "ballz",
			},
	}

	//! create mock DB and pass it to the fetcher
	mockDB := mocks.NewComicsDB(t)
	flushedComics := make([]*db.Comic, 2)
	mockDB.On("FlushBatch", mock.AnythingOfType("[]*db.Comic")).Return(nil).Run(func(args mock.Arguments){
		flushed := args.Get(0).([]*db.Comic) // copied the type assertion from documentation
		flushedComics = flushed
	})

	lg := &logger.Wood{Logger: log.New(os.Stderr, "TestFetchBatch", log.Lshortfile)}

	f := &Fetcher{
		db:     mockDB,
		logger: lg,

		url:       url,
		workers:   1,
		maxComics: len(expectedComics),
		batchSize: 2,
	}

	missingChan := make(chan int)
	commitedChan := make(chan int)
	cc := 0
	mc := 0
	var wg sync.WaitGroup

	wc := workConfig{
		completeWork: commitedChan,
		failedWork: missingChan,
		wg: &wg,
		b: &batch{start:1, end: 2},

	}
	/* initialization complete */

	/* start test */
	wg.Add(f.workers)
	go f.fetchBatch(&wc)

	go func() {
		for missing := range missingChan {
			mc += missing
		}
	}()
	go func() {
		for commited := range commitedChan {
			cc += commited
		}
	}()
	wg.Wait()
	close(missingChan)
	close(commitedChan)

	// flushed matches expected
	assert.Equal(t, expectedComics, flushedComics)
}

func TestDownload(t *testing.T) {
	// test downloading comics from offset 0

	// i can make a test server, and hook it to the fetcher
	// but how do i make sure that download() has done its job?
	// download() takes in an offset and returns number of comics downloaded,
	// internally it dispatches workers to download batches of comics.
	// it does the job right when it dispatches workers that cover the entire comics range
	// so basically i'm testing the for loop inside the download()

	// we can make sure the for loop is right when the server recieves requests for all the comic numbers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){

	}))

	defer ts.Close()


}

func TestDispatch(t *testing.T){
	// dispatch func requires a fetcher object, work func, configuration, offset.
	// it contains a concurrency pattern, workers pattern? how do we test that?
	f := &Fetcher{
		workers: 10,
		maxComics: 100,
	}
	lg := &logger.Wood{Logger: log.New(os.Stderr, "TestDispatch", log.Lshortfile)}
	wc := workConfig{

	}
	work := func(wc *workConfig){

	}
}

func TestSimpleFetch(t *testing.T){

}