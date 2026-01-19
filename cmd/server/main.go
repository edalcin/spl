package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed views/*
var views embed.FS

var db *sql.DB

type Item struct {
	ID        int
	Name      string
	Completed bool
}

func main() {
	// Configuração do Banco de Dados
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "shopping.db"
	}

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable()

	// Roteamento
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/toggle/", toggleHandler)
	http.HandleFunc("/delete/", deleteHandler)

	// Servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor rodando em http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		completed BOOLEAN NOT NULL DEFAULT 0
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	items, err := getItems()
	if err != nil {
		http.Error(w, "Erro ao buscar itens", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFS(views, "views/index.html")
	if err != nil {
		log.Printf("Erro template: %v", err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, map[string]interface{}{
		"Items": items,
	})
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método inválido", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	if strings.TrimSpace(name) != "" {
		_, err := db.Exec("INSERT INTO items (name, completed) VALUES (?, ?)", name, false)
		if err != nil {
			http.Error(w, "Erro ao salvar", http.StatusInternalServerError)
			return
		}
	}

	// Retorna apenas a lista atualizada para o HTMX
	items, _ := getItems()
	tmpl, _ := template.ParseFS(views, "views/index.html")
	tmpl.ExecuteTemplate(w, "item-list", items)
}

func toggleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/toggle/")
	id, _ := strconv.Atoi(idStr)

	var completed bool
	err := db.QueryRow("SELECT completed FROM items WHERE id = ?", id).Scan(&completed)
	if err == nil {
		db.Exec("UPDATE items SET completed = ? WHERE id = ?", !completed, id)
	}

	items, _ := getItems()
	tmpl, _ := template.ParseFS(views, "views/index.html")
	tmpl.ExecuteTemplate(w, "item-list", items)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/delete/")
	id, _ := strconv.Atoi(idStr)

	db.Exec("DELETE FROM items WHERE id = ?", id)

	items, _ := getItems()
	tmpl, _ := template.ParseFS(views, "views/index.html")
	tmpl.ExecuteTemplate(w, "item-list", items)
}

func getItems() ([]Item, error) {
	rows, err := db.Query("SELECT id, name, completed FROM items ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var i Item
		if err := rows.Scan(&i.ID, &i.Name, &i.Completed); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, nil
}
