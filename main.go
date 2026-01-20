package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Load the Postgres driver
)

func main() {
	// Load environment variables from .env file for local development
	godotenv.Load()

	// 1. Retrieve the Database URL from environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set!")
	}

	// 2. Establish a connection to the database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
	defer db.Close()

	// 3. Initialize the database schema
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS visits (
		id SERIAL PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	r := gin.Default()

	// 4. Configure templates and static file serving
	r.LoadHTMLGlob("templates/*")   // Points to your HTML files
	r.Static("/static", "./static") // Points to your CSS/JS files

	r.GET("/", func(c *gin.Context) {
		// 5. Register a new visit in the database
		_, err := db.Exec("INSERT INTO visits DEFAULT VALUES")
		if err != nil {
			c.String(http.StatusInternalServerError, "Internal Server Error: Could not save visit")
			return
		}

		// 6. Retrieve the total count of visits
		var total int
		err = db.QueryRow("SELECT COUNT(*) FROM visits").Scan(&total)
		if err != nil {
			c.String(http.StatusInternalServerError, "Internal Server Error: Could not retrieve data")
			return
		}

		// 7. Render the index.html template with data
		c.HTML(http.StatusOK, "index.html", gin.H{
			"total_visits": total,
		})
	})

	// 8. Start the server on the specified port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}

