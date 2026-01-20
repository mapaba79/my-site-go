package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func main() {
	// Carrega o .env apenas localmente. No Render ele vai ignorar se o arquivo não existir.
	godotenv.Load()

	// 1. Configuração do Postgres
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Erro na conexão com Postgres:", err)
	}
	defer db.Close()

	// 2. Configuração do Redis (Key-Value no Render)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Erro ao processar REDIS_URL: %v. Usando padrão localhost.", err)
		opt = &redis.Options{Addr: "localhost:6379"}
	}
	rdb := redis.NewClient(opt)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		// A. Salva a visita no Postgres (Sempre o registro oficial)
		_, err := db.Exec("INSERT INTO visits DEFAULT VALUES")
		if err != nil {
			log.Println("Erro ao inserir no Postgres:", err)
		}

		// B. Incrementa no Redis e já pega o novo valor
		newVal, err := rdb.Incr(ctx, "total_visits").Result()

		// C. Sincronização: Se o Redis retornou 1, pode ser que ele tenha resetado.
		// Vamos conferir no Postgres o total real.
		if newVal == 1 {
			var realCount int64
			err := db.QueryRow("SELECT COUNT(*) FROM visits").Scan(&realCount)
			if err == nil && realCount > 1 {
				rdb.Set(ctx, "total_visits", realCount, 0)
				newVal = realCount
				log.Println("Redis sincronizado com o total do Postgres")
			}
		}

		log.Printf("Visita registrada! Total: %d", newVal)

		c.HTML(http.StatusOK, "index.html", gin.H{
			"total_visits": newVal,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	r.Run(":" + port)
}

