package download

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ripp4rd0c/xkcd/db"
	"github.com/ripp4rd0c/xkcd/logger"
)

type Fetcher struct {
	db     db.ComicsDB
	logger *logger.Wood

	url       string
	workers   int
	maxComics int // maximum portion of comics needed to download
	batchSize int // size of batch given to each worker
}

func NewFetcher(cfg *Config) (*Fetcher, error) {

	workers := runtime.NumCPU() * 2

	// size of the batch each worker must download, so that by the end the sum of
	// downloaded batches equals _maxComics_
	batchSize := cfg.MaxComics / workers
	if batchSize < 1 {
		fmt.Println("made batch size 1")
		batchSize = 1
	}

	return &Fetcher{
		db:        cfg.Db,
		logger:    cfg.Logger,
		maxComics: cfg.MaxComics,
		url:       cfg.URL,

		workers:   workers,
		batchSize: batchSize,
	}, nil
}

// Download is a wrapper around dispatch()
// returns number of downloaded comics
func (f *Fetcher) Download(offset int) int {
	missingCount := 0
	commitedCount := 0
	missingChan := make(chan int)
	commitedChan := make(chan int)
	work := make(chan int)

	var wg sync.WaitGroup
	wg.Add(f.workers)

	go func() {
		// count missing comics
		for mc := range missingChan {
			missingCount += mc
		}
	}()

	go func() {
		// count successfully commited comics
		for cc := range commitedChan {
			commitedCount += cc
		}
	}()

	wc := &workConfig{
		wg:           &wg,
		work:         work,
		completeWork: commitedChan,
		failedWork:   missingChan,
	}

	f.dispatch(f.simpleFetch, wc, offset)

	// for i := 0; i < f.workers+1; i++ {
	// 	start := offset + (i*f.batchSize + 1)
	// 	end := offset + (i*f.batchSize+f.batchSize)
	// 	// don't exceed max comics
	// 	end = int(math.Min(float64(end), float64(f.maxComics)))

	// 	// start += offset
	// 	// end += offset //? what if this exceeds f.maxComics

	// 	f.logger.LogDebug(start, end)

	// 	//! send a worker function with channles as params to be ready to recv the comics range (work)
	// 	go f.fetchBatch(start, end, &wg, missingChan, commitedChan)
	// }

	// for bla := 0; bla < f.maxComics; {
	// 	work <- (bla, bla+250)
	// 	bla+=250
	// }

	wg.Wait()
	close(missingChan)
	close(commitedChan)

	f.logger.LogInfo(fmt.Sprintf("%d comics are missing", missingCount))

	return commitedCount - missingCount
}

// parameters for the work function
type workConfig struct {
	wg           *sync.WaitGroup
	work         chan int
	completeWork chan int
	failedWork   chan int
	b            *batch
}

// i can make both download functions (batch + simple) use the same config struct, but each takes what it needs from it.
// this way both can be plugged into the dispatch function.

type batch struct {
	start int
	end   int
}

// dispatch fetcher's workers to execute function from an offset
func (f *Fetcher) dispatch(worker func(*workConfig), wc *workConfig, offset int) {

	for i := 0; i < f.workers; i++ {
		go worker(wc)
		f.logger.LogDebug(fmt.Sprintf("worker %d up!", i))
	}

	for c := offset; c <= f.maxComics; c++ {
		wc.work <- c
	}
	close(wc.work)
}

// not concerned with offset, it just downloads comics.
func (f *Fetcher) simpleFetch(wc *workConfig) {
	defer wc.wg.Done()

	// maintain a buffer
	buffer := make([]*db.Comic, 0, 100)
	defer f.flushToDb(buffer)

	// get comic number from work channel
	for num := range wc.work {
		// download comic and append to buffer
		url := fmt.Sprintf(f.url, num)
		c, err := fetchComic(url, num)
		if err != nil {
			fmt.Println(err) //! fix this
			wc.failedWork <- 1
			continue
		}
		buffer = append(buffer, c)

		// check buffer is full --> commit to database & clear buffer
		if len(buffer) == cap(buffer) {
			f.flushToDb(buffer)
			// report number of flushed items
			wc.completeWork <- len(buffer)
			// clear buffer
			buffer = buffer[:0]
		}
	}
}

// TODO find a way to download a range more cleanly
func (f *Fetcher) fetchBatch(wc *workConfig) {
	defer wc.wg.Done()

	start, end := wc.b.start, wc.b.end

	f.logger.LogInfo(fmt.Sprintf("downloading comics range [%d, %d]", start, end))

	buffer := make([]*db.Comic, 0, 20) // length 0, capacity 20. avoid having init nil elements

	for i := start; i <= end; i++ {
		url := fmt.Sprintf(f.url, i)
		c, err := fetchComic(url, i)
		if err != nil {
			fmt.Println(err)
			wc.failedWork <- 1
			continue
		}
		buffer = append(buffer, c)

		// flush buffer
		if len(buffer) == 20 || i == end {
			// keep retrying until transaction success
			f.flushToDb(buffer)
			wc.completeWork <- len(buffer)

			// clear buffer
			buffer = buffer[:0]
		}
	}

}

func fetchComic(url string, num int) (*db.Comic, error) {
	fmt.Println("fetching: ", url)

	// request
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	//! handle 404
	if res.StatusCode > 400 {
		return nil, fmt.Errorf("missing comic number %d", num)
	}

	// parse body
	var c db.Comic
	jdata := json.NewDecoder(res.Body)
	jdata.Decode(&c) // do i really need to decode the bytes into json?

	return &c, nil
}

func (f *Fetcher) flushToDb(buffer []*db.Comic) {
	var err error
	// keep retrying until transaction success
	for err = f.db.FlushBatch(buffer); err != nil; err = f.db.FlushBatch(buffer) {
		f.logger.LogError("error flushing comic batch into DB")
		f.logger.LogError(err.Error())
		f.logger.LogError("retrying... *****")
		time.Sleep(200 * time.Millisecond)
	}
	f.logger.LogInfo(fmt.Sprintf("flushed comcis [%d, %d] into db", buffer[0].Num, buffer[len(buffer)-1].Num))
}

// func checkEndOfStream(stream404 <-chan int, done chan<- struct{}) {
// 	prev := 0
// 	counter := 0
// 	for {
// 		n := <-stream404
// 		if n > prev {
// 			if n == prev+1 {
// 				counter++
// 			}
// 			prev = n
// 		}
// 		if counter == 3 {
// 			break
// 		}
// 	}
// 	done <- struct{}{}
// }
