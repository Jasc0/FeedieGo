package main

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)
const DESC = false
const ASC = true

type timeOrder bool

var db *sql.DB
var err error
var dbMu sync.RWMutex
func ensureParentDirs(filePath string) error {
    dir := filepath.Dir(filePath)
    return os.MkdirAll(dir, os.ModePerm)
}

func DBInit(path string) {
	err = ensureParentDirs(path)
	if err != nil{
		log.Fatal(err)
	}
	dbMu.Lock()
	defer dbMu.Unlock()
	db, err = sql.Open("sqlite", fmt.Sprintf("file:%s",path))
	if err != nil{
		log.Fatal(err)
	}
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	if err != nil{
		log.Fatal(err)
	}
	_, err = db.Exec(`PRAGMA journal_mode=WAL`)
	if err != nil{
		log.Fatal(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS feeds (
		id TEXT PRIMARY KEY,
		title TEXT,
		url TEXT
	);`)
	if err != nil{
		log.Fatal(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS entries (
		id TEXT PRIMARY KEY,
		feed_id TEXT NOT NULL,
		title TEXT,
		author TEXT,
		published INTEGER,
		description TEXT,
		thumbnail TEXT,
		FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE
	);`)
	if err != nil{
		log.Fatal(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS links (
		id TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		entry_id TEXT NOT NULL,
		link_type TEXT,
		FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE
	);`)
	if err != nil{
		log.Fatal(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS tags (
		id TEXT PRIMARY KEY,
		name TEXT
	);`)
	if err != nil{
		log.Fatal(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS tag_members (
		tag_id TEXT NOT NULL,
		feed_id TEXT NOT NULL,
		PRIMARY KEY (tag_id, feed_id),
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
		FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE
	);`)
	if err != nil{
		log.Fatal(err)
	}
	db.SetMaxOpenConns(0)
}

func GetHashString(root string) string{
	hasher := fnv.New64a()
	hasher.Write([]byte(root))
	return fmt.Sprintf("%x", hasher.Sum64())
}

func DBExecuteSQL(statement string, args []any){
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err = db.Exec(statement, args...)
	if err != nil{
		log.Fatal(err)
	}
}

func DBAddFeed(feed FeedieFeed){
	hash := GetHashString(feed.Url)
	dbMu.Lock()
	defer dbMu.Unlock()
	statement :=`INSERT INTO feeds (id, title, url)
VALUES (?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    title = excluded.title,
    url = excluded.url;
`
	_, err = db.Exec(statement, hash, feed.Title, feed.Url)
	if err != nil{
		log.Fatal(err)
	}
}


func DBAddEntry(feed FeedieFeed, entry FeedieEntry){
	feed_id := GetHashString(feed.Url)
	entry_id := GetHashString(entry.getHashString())

	statement :=`INSERT INTO entries 
	(id, feed_id, title, author, published, description, thumbnail)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    feed_id = excluded.feed_id,
    title = excluded.title,
    author = excluded.author,
    published = excluded.published,
    description = excluded.description,
    thumbnail = excluded.thumbnail;
`
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err = db.Exec(statement,entry_id,feed_id,entry.Title,entry.Author,entry.Published, entry.Description, entry.Thumbnail)
	if err != nil{
		log.Fatal(err)
	}

	statement = `INSERT INTO links
	(id, url, entry_id, link_type)
	VALUES (?,?,?,?)
	ON CONFLICT(id) DO UPDATE SET
	entry_id = excluded.entry_id,
	link_type = excluded.link_type;`

	for _, link := range entry.Links{
		_, err = db.Exec(statement,GetHashString(link.URL+entry_id),link.URL,entry_id,link.Type)
		if err != nil{
			log.Fatal(err)
		}
	}
}

func DBAddFeedWithEntries(feed FeedieFeed){
	feed_id := GetHashString(feed.Url)

	dbMu.Lock()
	defer dbMu.Unlock()

	tx, err := db.Begin()
	if err != nil { log.Fatal(err) }

	_, err = tx.Exec(`INSERT INTO feeds (id, title, url)
VALUES (?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    title = excluded.title,
    url = excluded.url;`, feed_id, feed.Title, feed.Url)
	if err != nil { tx.Rollback(); log.Fatal(err) }

	for _, entry := range feed.Entries {
		entry_id := GetHashString(entry.getHashString())
		_, err = tx.Exec(`INSERT INTO entries
(id, feed_id, title, author, published, description, thumbnail)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    feed_id = excluded.feed_id,
    title = excluded.title,
    author = excluded.author,
    published = excluded.published,
    description = excluded.description,
    thumbnail = excluded.thumbnail;`,
			entry_id, feed_id, entry.Title, entry.Author, entry.Published, entry.Description, entry.Thumbnail)
		if err != nil { tx.Rollback(); log.Fatal(err) }

		for _, link := range entry.Links {
			_, err = tx.Exec(`INSERT INTO links (id, url, entry_id, link_type)
VALUES (?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
    entry_id = excluded.entry_id,
    link_type = excluded.link_type;`,
				GetHashString(link.URL+entry_id), link.URL, entry_id, link.Type)
			if err != nil { tx.Rollback(); log.Fatal(err) }
		}
	}

	if err = tx.Commit(); err != nil { log.Fatal(err) }
}

func DBAddTag(tag_name string){
	tag_id := GetHashString(tag_name)

	statement := `INSERT INTO tags
	(id, name)
	VALUES (?,?)
	ON CONFLICT(id) DO UPDATE SET
	name = excluded.name;`

	dbMu.Lock()
	defer dbMu.Unlock()
	_, err = db.Exec(statement,tag_id,tag_name)
	if err != nil{
		log.Fatal(err)
	}
}

func DBAddFeedToTag(feed FeedieFeed, tag string){
	var feed_id string
	var tag_id string

	dbMu.RLock()
	feedErr := db.QueryRow("SELECT id FROM feeds WHERE title = ? AND url = ?", feed.Title, feed.Url).Scan(&feed_id)
	tagErr  := db.QueryRow("SELECT id FROM tags WHERE name = ?", tag).Scan(&tag_id)
	dbMu.RUnlock()

	if feedErr != nil {
		if feedErr == sql.ErrNoRows {
			log.Println("No matching feed found")
			return
		}
		log.Fatal(feedErr)
	}
	if tagErr != nil {
		if tagErr == sql.ErrNoRows {
			log.Println("No matching tag found, creating it")
			DBAddTag(tag)
			DBAddFeedToTag(feed, tag)
			return
		}
		log.Fatal(tagErr)
	}

	dbMu.Lock()
	defer dbMu.Unlock()
	_, err = db.Exec(`INSERT INTO tag_members
	(tag_id, feed_id)
	VALUES (?,?);`, tag_id, feed_id)
	if err != nil{
		log.Fatal(err)
	}
}

// scanEntries aggregates a LEFT JOIN result (entries + links) into []FeedieEntry.
// Each entry may appear on multiple rows (one per link); NULL link columns mean no links.
func scanEntries(rows *sql.Rows) []FeedieEntry {
	ret := []FeedieEntry{}
	var cur *FeedieEntry
	var curID string
	for rows.Next() {
		var id, title, author, description, thumbnail string
		var published int64
		var linkURL, linkType sql.NullString
		err := rows.Scan(&id, &title, &author, &description, &thumbnail, &published, &linkURL, &linkType)
		if err != nil {
			log.Fatal(err)
		}
		if id != curID {
			if cur != nil {
				ret = append(ret, *cur)
			}
			cur = newEntry(title, author, published, description, thumbnail)
			curID = id
		}
		if linkURL.Valid {
			cur.Links = append(cur.Links, FeedieLink{URL: linkURL.String, Type: linkType.String})
		}
	}
	if cur != nil {
		ret = append(ret, *cur)
	}
	return ret
}

func DBGetAllTimeOrdered(isAsc timeOrder) []FeedieEntry{
	query := `
SELECT e.id, e.title, e.author, e.description, e.thumbnail, e.published,
       l.url, l.link_type
FROM entries e
LEFT JOIN links l ON l.entry_id = e.id
ORDER BY e.published DESC, e.id`
	if isAsc{
		query = strings.Replace(query, "DESC", "ASC", 1)
	}
	dbMu.RLock()
	defer dbMu.RUnlock()
	rows, err := db.Query(query)
	if err != nil{
		log.Fatal(err)
	}
	defer rows.Close()
	return scanEntries(rows)
}

func DBGetByTagTimeOrdered(tag string, isAsc timeOrder) []FeedieEntry{
	query := `
SELECT e.id, e.title, e.author, e.description, e.thumbnail, e.published,
       l.url, l.link_type
FROM entries AS e
JOIN feeds AS f ON e.feed_id = f.id
JOIN tag_members AS tm ON f.id = tm.feed_id
JOIN tags AS t ON tm.tag_id = t.id
LEFT JOIN links l ON l.entry_id = e.id
WHERE t.name = ?
ORDER BY e.published DESC, e.id`
	if isAsc{
		query = strings.Replace(query, "DESC", "ASC", 1)
	}
	dbMu.RLock()
	defer dbMu.RUnlock()
	rows, err := db.Query(query, tag)
	if err != nil{
		log.Fatal(err)
	}
	defer rows.Close()
	return scanEntries(rows)
}

func DBGetFeedByName(name string) FeedieFeed {
	query := `SELECT title, url FROM feeds WHERE title = ?`
	var title, url string
	dbMu.RLock()
	feedData := db.QueryRow(query, name)
	if err != nil{
		log.Fatal(err)
	}
	dbMu.RUnlock()
	err := feedData.Scan(&title, &url)
	if err != nil{
		if err == sql.ErrNoRows{
			return FeedieFeed{}
		}
		log.Fatal(err)
	}
	feed := FeedieFeed{Title: title, Url: url}

	return feed

}

func DBGetByFeedTimeOrdered(feed FeedieFeed, isAsc timeOrder) []FeedieEntry{
	feed_id := GetHashString(feed.Url)
	query := `
SELECT e.id, e.title, e.author, e.description, e.thumbnail, e.published,
       l.url, l.link_type
FROM entries AS e
JOIN feeds AS f ON e.feed_id = f.id
LEFT JOIN links l ON l.entry_id = e.id
WHERE f.id = ?
ORDER BY e.published DESC, e.id`
	if isAsc{
		query = strings.Replace(query, "DESC", "ASC", 1)
	}
	dbMu.RLock()
	defer dbMu.RUnlock()
	rows, err := db.Query(query, feed_id)
	if err != nil{
		log.Fatal(err)
	}
	defer rows.Close()
	return scanEntries(rows)
}

func DBGetFeeds(withEntries bool ) []FeedieFeed{
	ret := []FeedieFeed{}
	query := `SELECT title, url FROM feeds`
	dbMu.RLock()
	defer dbMu.RUnlock()
	feeds, err := db.Query(query)
	if err != nil{
		log.Fatal(err)
	}
	defer feeds.Close()
	for feeds.Next() {
		if err != nil{
			log.Fatal(err)
		}
		var title, url string
		err = feeds.Scan(&title, &url)
		if err != nil{
			log.Fatal(err)
		}
		feed := FeedieFeed{Title: title, Url: url}
		if withEntries{
			ents := DBGetByFeedTimeOrdered(feed, DESC);
			feed.Entries = ents
		}
		ret = append(ret, feed)
	}
	return ret
}
func DBGetTags() []string{
	ret := []string{}
	query := `SELECT name FROM tags`
	dbMu.RLock()
	defer dbMu.RUnlock()
	feeds, err := db.Query(query)
	if err != nil{
		log.Fatal(err)
	}
	defer feeds.Close()
	for feeds.Next() {
		if err != nil{
			log.Fatal(err)
		}
		var tag string
		err = feeds.Scan(&tag)
		if err != nil{
			log.Fatal(err)
		}
		ret = append(ret, tag)
	}
	return ret
}

func DBDelTag(tag string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err = db.Exec(`
	DELETE FROM tags WHERE name = ?`, tag)
	if err != nil{
		log.Fatal(err)
	}
}

func DBDelFeed(feed_url string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	id := GetHashString(feed_url)
	_, err = db.Exec(`
	DELETE FROM feeds WHERE id = ?`, id)
	if err != nil{
		log.Fatal(err)
	}
}
func DBClearMembersTag(tag_name string){
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err = db.Exec(`DELETE FROM tag_members
	WHERE tag_id IN (
		SELECT id FROM tags WHERE name = ?
	);
	`, tag_name)
	if err != nil{
		log.Fatal(err)
	}
}
func DBAddMembership(tag_name, feed_url string){
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err = db.Exec(`INSERT INTO tag_members (tag_id, feed_id)
	VALUES (
		(SELECT id FROM tags WHERE name = ? LIMIT 1),
		(SELECT id FROM feeds WHERE url = ? LIMIT 1)
	);`, tag_name, feed_url)
	if err != nil{
		log.Fatal(err)
	}
}
// inverted refers to the query being "inverted" i.e. all feeds not in tag
func DBGetFeedsByTag(tag_name string, inverted bool) []FeedieFeed{
	dbMu.RLock()
	defer dbMu.RUnlock()
	ret := []FeedieFeed{}
	var query string
	if inverted{
		query = `SELECT f.title, f.url
		FROM feeds f
		WHERE NOT EXISTS (
			SELECT 1
			FROM tag_members tm
			JOIN tags t ON tm.tag_id = t.id
			WHERE tm.feed_id = f.id
			AND t.name = ?
		);
		`
	} else{
		query = `SELECT f.title, f.url
		FROM feeds f
		JOIN tag_members tm ON f.id = tm.feed_id
		JOIN tags t ON tm.tag_id = t.id
		WHERE t.name = ?;
		`
	}
	feeds, err := db.Query(query, tag_name)
	if err != nil{
		log.Fatal(err)
	}
	defer feeds.Close()
	for feeds.Next() {
		var title, url string
		err = feeds.Scan(&title, &url)
		if err != nil{
			log.Fatal(err)
		}
		feed := FeedieFeed{Title: title, Url: url}
		ret = append(ret, feed)
	}
	return ret
}
func shutDownDB() {
	if db != nil{
		db.Close()
	}
}
