package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // O '_' serve para carregar o driver sem chamá-lo diretamente
)

func main() {
	// 1. Pega a URL do banco das variáveis de ambiente
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("A variável DATABASE_URL não está configurada!")
	}

	// 2. Abre a conexão com o banco
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Erro ao conectar ao banco:", err)
	}
	defer db.Close()

	// 3. Cria uma tabela de teste se ela não existir
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS visitas (
		id SERIAL PRIMARY KEY,
		data_hora TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal("Erro ao criar tabela:", err)
	}

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		// 4. Insere uma nova visita toda vez que alguém acessa a rota "/"
		_, err := db.Exec("INSERT INTO visitas DEFAULT VALUES")
		if err != nil {
			c.String(http.StatusInternalServerError, "Erro ao salvar no banco")
			return
		}

		// 5. Conta o total de visitas para exibir na tela
		var total int
		err = db.QueryRow("SELECT COUNT(*) FROM visitas").Scan(&total)
		if err != nil {
			c.String(http.StatusInternalServerError, "Erro ao ler do banco")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":        "online",
			"mensagem":      "Visita registrada no Postgres!",
			"total_visitas": total,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	r.Run(":" + port)
}

