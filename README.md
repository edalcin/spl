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

## Funcionalidades

### 🔐 Autenticação
- **PIN Opcional:** Configure um PIN via variável de ambiente `APP_PIN` para proteger o acesso
- **Sessões Seguras:** Sistema de sessões com cookies HttpOnly e expiração automática (24h)
- **Renovação Automática:** Sessões renovadas a cada requisição (sliding expiration)
- **Acesso Público:** Se nenhum PIN for configurado, o app funciona sem autenticação

### 📋 Gerenciamento de Listas
- **Múltiplas Listas:** Crie e organize quantas listas de compras precisar
- **CRUD Completo:** Adicione, edite e exclua listas conforme necessário
- **Reordenar por Arraste:** Arraste listas pela alça (≡) na tela de gerenciamento para definir a ordem
- **Lista Padrão:** Uma "Lista Principal" é criada automaticamente no primeiro uso
- **Proteção:** Sistema impede a exclusão da última lista restante
- **Navegação Simplificada:** Alterne entre listas facilmente pela interface

### ✅ Gerenciamento de Itens
- **Adicionar Itens:** Insira produtos rapidamente na lista ativa
- **Edição Inline:** Clique no nome do item para editá-lo diretamente
- **Excluir Itens:** Remova itens indesejados com um clique
- **Reordenar por Arraste:** Arraste itens pela alça (≡) para reorganizar a ordem
- **Mover entre Listas:** Use o botão (→) para transferir um item para outra lista
- **Interface Reativa:** Atualização instantânea via HTMX sem recarregar a página

### 🧠 Memória e Autocomplete
- **Memória Inteligente:** O sistema memoriza todos os itens já adicionados em cada lista
- **Autocomplete Contextual:** Ao digitar, receba sugestões baseadas no histórico da lista atual
- **Busca Fuzzy:** Encontre itens mesmo digitando apenas parte do nome
- **Sugestões em Tempo Real:** Lista de até 10 sugestões aparece enquanto você digita
- **Seleção Rápida:** Clique na sugestão para preencher automaticamente

### ⚡ Performance & UX
- **Drag & Drop:** Reordenação intuitiva via SortableJS, com suporte a touch/mobile
- **Sem Build Frontend:** Interface HTMX + Pico.css, sem complexidade de build
- **Navegação Fluida:** Experiência SPA-like sem JavaScript pesado
- **Totalmente em Português:** Interface 100% localizada
- **Mobile-First:** Design responsivo que funciona em qualquer dispositivo

## Estrutura

- `cmd/server/main.go`: Lógica do servidor e banco de dados.
- `cmd/server/views/`: Templates HTML.
- `Dockerfile`: Receita de construção da imagem.