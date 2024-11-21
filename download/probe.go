package download

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ripp4rd0c/xkcd/db"
)

func FindLastComic() (int, error) {
	// TODO handle the case where curr is 404 but not the last
	var curr int
	for curr = 1; numberExists(curr); curr <<= 1 {
		fmt.Println(curr)
	}
	start := curr >> 1
	return binarySearch(start), nil
}

// finds first 404 comic in the range [l, l*2-1]
func binarySearch(l int) int {
	r := l*2 - 1
	var mid int

	for l <= r {
		mid = l + (r-l)/2

		if numberExists(mid) {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	return mid
}

// performs a HEAD request on the comic number, returns false if 404 is found.
func numberExists(i int) bool {
	url := fmt.Sprintf("https://xkcd.com/%d/info.0.json", i)
	fmt.Println("trying, ", url)

	res, err := http.Head(url)
	if err != nil {
		fmt.Println("error in the head request")
	}

	defer res.Body.Close()

	fmt.Println(res.StatusCode)

	return res.StatusCode == http.StatusOK
}

func LatestXkcd() (int, error){
	url := "http://xkcd.com/info.0.json"
	res, err := http.Get(url)
	if err!=nil {
		return -1, err
	}
	defer res.Body.Close()

	var c db.Comic
	jd := json.NewDecoder(res.Body)
	if err := jd.Decode(&c); err!=nil {
		return -1, err
	}
	
	return c.Num, nil
}