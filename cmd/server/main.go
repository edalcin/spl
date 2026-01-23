package main

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/base64"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed views static
var views embed.FS

var (
	db       *sql.DB
	appPIN   string
	sessions sync.Map // Thread-safe map for sessions [token]expiryTime
)

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
	Suggestions []string
	ShowManager bool
}

func main() {
	// Configuração do banco de dados
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./shopping.db"
	}

	// Configuração do PIN
	appPIN = os.Getenv("APP_PIN")

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()

	// Arquivos Estáticos
	http.Handle("/static/", http.FileServer(http.FS(views)))

	// Handler de Login (Público)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)

	// Rotas Protegidas
	http.HandleFunc("/", protected(indexHandler))
	http.HandleFunc("/manage", protected(manageHandler))

	// Listas
	http.HandleFunc("/list/", protected(listViewHandler))
	http.HandleFunc("/lists/add", protected(createListHandler))
	http.HandleFunc("/lists/edit/", protected(editListHandler))
	http.HandleFunc("/lists/delete/", protected(deleteListHandler))

	// Itens
	http.HandleFunc("/items/add", protected(addItemHandler))
	http.HandleFunc("/items/edit/", protected(editItemHandler))
	http.HandleFunc("/items/delete/", protected(deleteItemHandler))
	http.HandleFunc("/items/forget/", protected(forgetItemHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor rodando em http://localhost:%s (PIN Ativado: %v)", port, appPIN != "")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// --- Auth Middleware & Handlers ---

func protected(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if appPIN == "" {
			next(w, r)
			return
		}

		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		expiry, ok := sessions.Load(cookie.Value)
		if !ok || time.Now().After(expiry.(time.Time)) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		sessions.Store(cookie.Value, time.Now().Add(24*time.Hour))

		next(w, r)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, _ := template.ParseFS(views, "views/login.html")
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		pin := r.FormValue("pin")
		if pin == appPIN {
			token := generateToken()
			sessions.Store(token, time.Now().Add(24*time.Hour))

			http.SetCookie(w, &http.Cookie{
				Name:     "session_token",
				Value:    token,
				Expires:  time.Now().Add(24 * time.Hour),
				HttpOnly: true,
				Path:     "/",
			})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		tmpl, _ := template.ParseFS(views, "views/login.html")
		tmpl.Execute(w, map[string]string{"Error": "PIN Incorreto"})
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessions.Delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session_token",
		Value:  "",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// --- Database Logic ---

func initDB() {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS lists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`)
	if err != nil {
		log.Fatal(err)
	}

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

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS item_memory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		list_id INTEGER,
		name TEXT NOT NULL,
		FOREIGN KEY(list_id) REFERENCES lists(id) ON DELETE CASCADE
	);`)
	if err != nil {
		log.Fatal(err)
	}

	db.Exec(`ALTER TABLE items ADD COLUMN list_id INTEGER DEFAULT 1;`)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM lists").Scan(&count)
	if count == 0 {
		res, _ := db.Exec("INSERT INTO lists (name) VALUES (?)", "Lista Principal")
		id, _ := res.LastInsertId()
		db.Exec("UPDATE items SET list_id = ? WHERE list_id IS NULL OR list_id = 0", id)
	}
}

// --- App Handlers ---

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var firstListID int
	err := db.QueryRow("SELECT id FROM lists ORDER BY id ASC LIMIT 1").Scan(&firstListID)
	if err == nil {
		http.Redirect(w, r, "/list/"+strconv.Itoa(firstListID), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/manage", http.StatusSeeOther)
}

func manageHandler(w http.ResponseWriter, r *http.Request) {
	data := getDataForList(0)
	data.ShowManager = true
	renderView(w, data)
}

func listViewHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/list/")
	listID, _ := strconv.Atoi(idStr)
	renderView(w, getDataForList(listID))
}

func createListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	name := r.FormValue("name")
	if strings.TrimSpace(name) != "" {
		db.Exec("INSERT INTO lists (name) VALUES (?)", name)
	}
	http.Redirect(w, r, "/manage", http.StatusSeeOther)
}

func editListHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/lists/edit/")
	id, _ := strconv.Atoi(idStr)
	newName := r.FormValue("name")
	if strings.TrimSpace(newName) != "" {
		db.Exec("UPDATE lists SET name = ? WHERE id = ?", newName, id)
	}
	http.Redirect(w, r, "/manage", http.StatusSeeOther)
}

func deleteListHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/lists/delete/")
	id, _ := strconv.Atoi(idStr)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM lists").Scan(&count)
	if count > 1 {
		db.Exec("DELETE FROM items WHERE list_id = ?", id)
		db.Exec("DELETE FROM item_memory WHERE list_id = ?", id)
		db.Exec("DELETE FROM lists WHERE id = ?", id)
	}
	http.Redirect(w, r, "/manage", http.StatusSeeOther)
}

func addItemHandler(w http.ResponseWriter, r *http.Request) {
	listID, _ := strconv.Atoi(r.FormValue("list_id"))
	name := r.FormValue("name")
	if strings.TrimSpace(name) != "" {
		db.Exec("INSERT INTO items (list_id, name, completed) VALUES (?, ?, ?)", listID, name, false)
		db.Exec("DELETE FROM item_memory WHERE list_id = ? AND name = ?", listID, name)
	}
	tmpl, _ := template.ParseFS(views, "views/index.html")
	tmpl.ExecuteTemplate(w, "items-partial", getDataForList(listID))
}

func editItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/items/edit/")
	id, _ := strconv.Atoi(idStr)
	newName := r.FormValue("name")
	if strings.TrimSpace(newName) != "" {
		var listID int
		db.QueryRow("SELECT list_id FROM items WHERE id = ?", id).Scan(&listID)
		db.Exec("UPDATE items SET name = ? WHERE id = ?", newName, id)
		tmpl, _ := template.ParseFS(views, "views/index.html")
		tmpl.ExecuteTemplate(w, "items-partial", getDataForList(listID))
	}
}

func deleteItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/items/delete/")
	id, _ := strconv.Atoi(idStr)
	var listID int
	var name string
	db.QueryRow("SELECT list_id, name FROM items WHERE id = ?", id).Scan(&listID, &name)
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM item_memory WHERE list_id = ? AND name = ?", listID, name).Scan(&exists)
	if exists == 0 {
		db.Exec("INSERT INTO item_memory (list_id, name) VALUES (?, ?)", listID, name)
	}
	db.Exec("DELETE FROM items WHERE id = ?", id)
	tmpl, _ := template.ParseFS(views, "views/index.html")
	tmpl.ExecuteTemplate(w, "items-partial", getDataForList(listID))
}

func forgetItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/items/forget/")
	id, _ := strconv.Atoi(idStr)
	var listID int
	var name string
	db.QueryRow("SELECT list_id, name FROM items WHERE id = ?", id).Scan(&listID, &name)
	db.Exec("DELETE FROM item_memory WHERE list_id = ? AND name = ?", listID, name)
	tmpl, _ := template.ParseFS(views, "views/index.html")
	tmpl.ExecuteTemplate(w, "items-partial", getDataForList(listID))
}

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
	var items []Item
	var suggestions []string
	if listID > 0 {
		db.QueryRow("SELECT id, name FROM lists WHERE id = ?", listID).Scan(&currentList.ID, &currentList.Name)
		iRows, _ := db.Query("SELECT id, list_id, name, completed FROM items WHERE list_id = ? ORDER BY completed ASC, id DESC", listID)
		defer iRows.Close()
		for iRows.Next() {
			var i Item
			iRows.Scan(&i.ID, &i.ListID, &i.Name, &i.Completed)
			items = append(items, i)
		}
		sRows, _ := db.Query("SELECT DISTINCT name FROM item_memory WHERE list_id = ? ORDER BY name", listID)
		defer sRows.Close()
		for sRows.Next() {
			var name string
			sRows.Scan(&name)
			suggestions = append(suggestions, name)
		}
	}
	return PageData{Lists: lists, CurrentList: currentList, Items: items, Suggestions: suggestions, ShowManager: false}
}

func renderView(w http.ResponseWriter, data PageData) {
	tmpl, err := template.ParseFS(views, "views/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}
