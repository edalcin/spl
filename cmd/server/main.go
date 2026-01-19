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

type List struct {
	ID   int
	Name string
}

type Item struct {
	ID        int
	ListID    int
	Name      string
	Completed bool
}

type PageData struct {
	Lists       []List
	CurrentList List
	Items       []Item
}

func main() {
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

	initDB()

	// Rotas de Listas
	http.HandleFunc("/", indexHandler)          // Redireciona ou mostra Home
	http.HandleFunc("/list/", listViewHandler)  // Ver uma lista específica
	http.HandleFunc("/lists/add", createListHandler)
	http.HandleFunc("/lists/edit/", editListHandler)
	http.HandleFunc("/lists/delete/", deleteListHandler)

	// Rotas de Itens
	http.HandleFunc("/items/add", addItemHandler)
	http.HandleFunc("/items/toggle/", toggleItemHandler)
	http.HandleFunc("/items/delete/", deleteItemHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor rodando em http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() {
	// Tabela de Listas
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS lists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`)
	if err != nil {
		log.Fatal(err)
	}

	// Tabela de Itens
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		list_id INTEGER,
		name TEXT NOT NULL,
		completed BOOLEAN NOT NULL DEFAULT 0,
		FOREIGN KEY(list_id) REFERENCES lists(id) ON DELETE CASCADE
	);`)
	if err != nil {
		log.Fatal(err)
	}

	// Migração simples: Se list_id não existe (versão antiga), adiciona
	// SQLite não suporta "IF NOT EXISTS" em ADD COLUMN facilmente, então ignoramos erro
	db.Exec(`ALTER TABLE items ADD COLUMN list_id INTEGER DEFAULT 1;`)

	// Garante que exista pelo menos uma lista se a tabela estiver vazia
	var count int
	db.QueryRow("SELECT COUNT(*) FROM lists").Scan(&count)
	if count == 0 {
		res, _ := db.Exec("INSERT INTO lists (name) VALUES (?)", "Lista Principal")
		id, _ := res.LastInsertId()
		// Atualiza itens antigos órfãos para a nova lista
		db.Exec("UPDATE items SET list_id = ? WHERE list_id IS NULL OR list_id = 0", id)
	}
}

// --- Handlers de Lista ---

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Pega a primeira lista e redireciona
	var firstListID int
	err := db.QueryRow("SELECT id FROM lists ORDER BY id ASC LIMIT 1").Scan(&firstListID)
	if err == nil {
		http.Redirect(w, r, "/list/"+strconv.Itoa(firstListID), http.StatusSeeOther)
		return
	}
	// Se não tem listas (não deveria acontecer devido ao initDB), cria uma view vazia
	renderView(w, PageData{})
}

func listViewHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/list/")
	listID, _ := strconv.Atoi(idStr)

	renderView(w, getDataForList(listID))
}

func createListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { return }
	name := r.FormValue("name")
	if strings.TrimSpace(name) != "" {
		res, _ := db.Exec("INSERT INTO lists (name) VALUES (?)", name)
		id, _ := res.LastInsertId()
		http.Redirect(w, r, "/list/"+strconv.Itoa(int(id)), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func editListHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/lists/edit/")
	id, _ := strconv.Atoi(idStr)
	newName := r.FormValue("name")

	if strings.TrimSpace(newName) != "" {
		db.Exec("UPDATE lists SET name = ? WHERE id = ?", newName, id)
	}
	http.Redirect(w, r, "/list/"+strconv.Itoa(id), http.StatusSeeOther)
}

func deleteListHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/lists/delete/")
	id, _ := strconv.Atoi(idStr)

	// Exclui lista e itens (Cascade pode não estar ativado por padrão no driver, então garantimos)
	db.Exec("DELETE FROM items WHERE list_id = ?", id)
	db.Exec("DELETE FROM lists WHERE id = ?", id)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// --- Handlers de Itens ---

func addItemHandler(w http.ResponseWriter, r *http.Request) {
	listID, _ := strconv.Atoi(r.FormValue("list_id"))
	name := r.FormValue("name")

	if strings.TrimSpace(name) != "" {
		db.Exec("INSERT INTO items (list_id, name, completed) VALUES (?, ?, ?)", listID, name, false)
	}

	// Retorna apenas a lista de itens atualizada (HTMX)
	tmpl, _ := template.ParseFS(views, "views/index.html")
	tmpl.ExecuteTemplate(w, "items-partial", getDataForList(listID))
}

func toggleItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/items/toggle/")
	id, _ := strconv.Atoi(idStr)

	var completed bool
	var listID int
	db.QueryRow("SELECT completed, list_id FROM items WHERE id = ?", id).Scan(&completed, &listID)
	db.Exec("UPDATE items SET completed = ? WHERE id = ?", !completed, id)

	tmpl, _ := template.ParseFS(views, "views/index.html")
	tmpl.ExecuteTemplate(w, "items-partial", getDataForList(listID))
}

func deleteItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/items/delete/")
	id, _ := strconv.Atoi(idStr)

	var listID int
	db.QueryRow("SELECT list_id FROM items WHERE id = ?", id).Scan(&listID)
	db.Exec("DELETE FROM items WHERE id = ?", id)

	tmpl, _ := template.ParseFS(views, "views/index.html")
	tmpl.ExecuteTemplate(w, "items-partial", getDataForList(listID))
}

// --- Helpers ---

func getDataForList(listID int) PageData {
	lists := []List{}
	rows, _ := db.Query("SELECT id, name FROM lists ORDER BY id")
	defer rows.Close()
	for rows.Next() {
		var l List
		rows.Scan(&l.ID, &l.Name)
		lists = append(lists, l)
	}

	var currentList List
	db.QueryRow("SELECT id, name FROM lists WHERE id = ?", listID).Scan(&currentList.ID, &currentList.Name)

	items := []Item{}
	iRows, _ := db.Query("SELECT id, list_id, name, completed FROM items WHERE list_id = ? ORDER BY completed ASC, id DESC", listID)
	defer iRows.Close()
	for iRows.Next() {
		var i Item
		iRows.Scan(&i.ID, &i.ListID, &i.Name, &i.Completed)
		items = append(items, i)
	}

	return PageData{Lists: lists, CurrentList: currentList, Items: items}
}

func renderView(w http.ResponseWriter, data PageData) {
	tmpl, err := template.ParseFS(views, "views/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}