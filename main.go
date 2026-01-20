package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func main() {
	godotenv.Load()

	// 1. Database Connection (Postgres)
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Postgres connection error:", err)
	}
	defer db.Close()

	// 2. Cache Connection (Redis)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Redis URL error:", err)
	}
	rdb := redis.NewClient(opt)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		// A. Save the visit in Postgres (The source of truth)
		_, err := db.Exec("INSERT INTO visits DEFAULT VALUES")
		if err != nil {
			log.Println("Postgres Insert Error:", err)
		}

		// B. Try to get and increment the count in Redis
		// If the key doesn't exist, we fetch from DB and save to Redis
		val, err := rdb.Get(ctx, "total_visits").Int()

		if err == redis.Nil {
			// Cache Miss: Redis is empty, ask Postgres
			log.Println("Cache Miss! Fetching count from Postgres...")
			err = db.QueryRow("SELECT COUNT(*) FROM visits").Scan(&val)
			if err != nil {
				c.String(http.StatusInternalServerError, "Database Error")
				return
			}
			// Store in Redis for 10 minutes to keep it fresh
			rdb.Set(ctx, "total_visits", val, 10*time.Minute)
		} else {
			// Cache Hit: Just increment the number in Redis
			log.Println("Cache Hit! Using Redis data...")
			val++ // Increment local variable for display
			rdb.Incr(ctx, "total_visits")
		}

		// C. Render the HTML
		c.HTML(http.StatusOK, "index.html", gin.H{
			"total_visits": val,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("App running on port %s", port)
	r.Run(":" + port)
}

