package db

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	Table = "comics"
)

var (
	ErrConn = errors.New("connection to database failed")
	ErrQuery= errors.New("query execution failed")
	ErrExec = errors.New("command execution on database failed")
	ErrTrans = errors.New("database transaction failed")
)

type Comic struct {
	Num   int    `json:"num"`
	Alt   string `json:"alt"`
	Img   string `json:"img"`
	Title string `json:"title"`
}

// DB implements the ComicsDB interface, and has access to sql.DB methods.
type DB struct {
	*sql.DB
}
//go:generate mockery --name ComicsDB
type ComicsDB interface {
	FlushBatch([]*Comic)error
	QueryComics(...string)error
	LastComic() (int, error)
}

func InitDB() (*DB, error) {
	db, err := sql.Open("sqlite3", "/home/alice/projects/xkcd/xkcd.db")
	if err != nil {
		return nil, ErrConn
	}
	// use a write-ahead log for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		return nil, ErrExec
	}
	// create comics table
	db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s using fts5(num, alt, img, title);", Table))

	return &DB{
		db,
	}, nil
}

// FlushBatch takes in a list of pointers to Comic structs, and performs transaction to
// push them to database. returns an error or nil.
func (db *DB) FlushBatch(comics []*Comic) error {
	// use a database transaction to commit the comic batch for more effeciency
	tx, err := db.Begin()
	if err != nil {
		return ErrTrans
	}
	defer tx.Rollback()
	stmt, err := db.Prepare(fmt.Sprintf("insert into %s values(?, ?, ?, ?);", Table))
	if err != nil {
		return err
	}

	for c := 0; c < len(comics); c++ {
		num, alt, img, title := comics[c].Num, comics[c].Alt, comics[c].Img, comics[c].Title
		_, err := tx.Stmt(stmt).Exec(num, alt, img, title)
		if err != nil {
			fmt.Printf("database transaction failed at %d\n", num)
			return ErrTrans
		}
	}
	if err := tx.Commit(); err != nil {
		fmt.Printf("transaction commit failed")
		return ErrTrans
	}

	return nil
}

func (db *DB) QueryComics(keywords ...string) error{
	/*	
	-- Query for all rows that contain at least once instance of the term
	-- "fts5" (in any column). Return results in order from best to worst
	-- match.
	"SELECT * FROM email WHERE email MATCH 'fts5' ORDER BY rank;"
	*/
	// sql := "SELECT * FROM comics WHERE comics MATCH (group_concat(,'OR')) ORDER BY rank;"

	if len(keywords) == 0 {
		return fmt.Errorf("you didn't provide any keywords")
	}

	//! sanitize input: only allow alphanumeric characters
	for i, v  := range keywords{
		keywords[i] = sanitize(v)
		// keywords[i]+= "*"
	}

	match := strings.Join(keywords, " OR ")

	sql := fmt.Sprintf("SELECT * FROM comics WHERE comics MATCH '%s' order by rank;", match)
	fmt.Println(sql)


	kargs := make([]interface{}, len(keywords))
	for i, v := range keywords {
		kargs[i] = v
	}
	fmt.Println(kargs...)

	rows, err := db.Query(sql, kargs...)
	if err!=nil{
		fmt.Println(err.Error())
		return fmt.Errorf("rows error")
	}

	for  rows.Next() {
		// fmt.Printf("rows: %v\n", rows)
		var num, alt, img, title string
		rows.Scan(&num, &alt, &img, &title)
		fmt.Printf("%s | %s | %s\n", title, num, img)
	}
	
	return nil
}

func sanitize(keyword string) string{
	re := regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	return re.ReplaceAllLiteralString(keyword, "")
}

func (db *DB) LastComic() (int, error){
	row := db.QueryRow("SELECT num FROM comics ORDER BY num DESC LIMIT 1;")
	var n int	
	if err:= row.Scan(&n); err!=nil{
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return -1, err
	}

	return n, nil
}