package main

import (
	"database/sql"

	"fmt"

	"hash/crc32"

	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // PostgreSQL driver
)

var (
	host = os.Getenv("DB_HOST")
	port = os.Getenv("DB_PORT")
	user = os.Getenv("DB_USER")
	password = os.Getenv("DB_PASSWORD")
	dbname = os.Getenv("DB_NAME")
)

func getLongLink(c *gin.Context) {
	shortURL := c.Param("short_url")

	if shortURL == "" {
		c.JSON(400, gin.H{"error": "short_url is required"})
		return
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		c.JSON(500, gin.H{"error": "database connection failed"})
		return
	}
	defer db.Close()

	var longURL string
	err = db.QueryRow("SELECT long_url FROM url WHERE short_url = $1", shortURL).Scan(&longURL)
	if err == sql.ErrNoRows {
		c.JSON(404, gin.H{"error": "short_url not found"})
		return
	} else if err != nil {
		c.JSON(500, gin.H{"error": "database query failed"})
		return
	}

	c.JSON(200, gin.H{"long_url": longURL})
}

func importHash(s string) string {
	// Hash with crc32
	return fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(s)))
}

func postLongLink(c *gin.Context) {
	longURL := c.PostForm("long_url")
	if longURL == "" {
		c.JSON(400, gin.H{"error": "long_url is required"})
		return
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		c.JSON(500, gin.H{"error": "database connection failed"})
		return
	}
	defer db.Close()

	// Check if longURL already exists
	var existingShortURL string
	err = db.QueryRow("SELECT short_url FROM url WHERE long_url = $1", longURL).Scan(&existingShortURL)
	if err == nil {
		// longURL exists, return its short_url
		c.JSON(200, gin.H{"short_url": existingShortURL})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(500, gin.H{"error": "database query failed"})
		return
	}

	// take only the first 7 characters of the hash
	shortURL := importHash(longURL)[:7]
	_, err = db.Exec("INSERT INTO url (short_url, long_url) VALUES ($1, $2)", shortURL, longURL)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("database insert failed: %v", err)})
		return
	}

	c.JSON(200, gin.H{"short_url": shortURL})
}

func getHomePage(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Welcome to Shawty URL Shortener!"})
}

func main(){
	router:=gin.Default()
	router.GET("", getHomePage)
	router.GET("/long/:short_url", getLongLink)
	router.POST("/long", postLongLink)
	router.Run("localhost:8080")
}