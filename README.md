# Simple Purchase List (SPL)

Um aplicativo de lista de compras minimalista, todo em Português, focado em simplicidade e performance.

Baseado na arquitetura do [Koffan](https://github.com/PanSalut/Koffan).

## Tecnologias

- **Backend:** Go (Golang)
- **Frontend:** HTMX + Pico.css (Sem build de frontend complexo)
- **Banco de Dados:** SQLite (Embarcado)
- **Docker:** Multi-stage build (~15-20MB imagem final)

## Como Rodar

### Com Docker (Recomendado)

1. **Construir a imagem:**
   ```bash
   docker build -t ghcr.io/edalcin/spl:latest .
   ```

2. **Rodar o container (com PIN opcional):**
   ```bash
   docker run -d -p 8080:8080 -v spl_data:/data -e APP_PIN=1234 ghcr.io/edalcin/spl:latest
   ```

3. Acesse `http://localhost:8080`

### Desenvolvimento Local

Requer Go 1.21+.

```bash
# Na raiz do projeto
go mod tidy
go run cmd/server/main.go
```

## Estrutura

- `cmd/server/main.go`: Lógica do servidor e banco de dados.
- `cmd/server/views/`: Templates HTML.
- `Dockerfile`: Receita de construção da imagem.