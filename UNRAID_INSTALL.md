# Instalação no Unraid (Docker)

Siga estes passos para instalar o **SPL (Simple Purchase List)** no seu servidor Unraid.

## Pré-requisitos
- Um servidor Unraid rodando.
- Docker habilitado.

## Passo a Passo (Instalação Manual)

1. Vá para a aba **Docker** no painel do Unraid.
2. Clique em **ADD CONTAINER** (botão na parte inferior).
3. Ative o **Advanced View** (interruptor no canto superior direito), se desejar personalizar mais detalhes, mas o básico funciona no modo simples.
4. Preencha os campos da seguinte forma:

| Campo | Valor | Notas |
| :--- | :--- | :--- |
| **Name** | `SPL` | Ou qualquer nome que preferir. |
| **Repository** | `ghcr.io/edalcin/spl:latest` | A imagem oficial do projeto. |
| **Network Type** | `Bridge` | Padrão. |
| **Console shell command** | `Shell` | Padrão. |
| **WebUI** | `http://[IP]:[PORT:8080]` | Permite clicar no ícone para abrir. |

5. **Adicione as Configurações de Porta e Caminho:**

   **Porta (Container Port: 8080):**
   - Clique em **Add another Path, Port, Variable, Label or Device**.
   - **Config Type:** Port
   - **Name:** Web Port
   - **Host Port:** `8080` (Ou outra porta livre, ex: `9090`)
   - **Container Port:** `8080` (NÃO ALTERAR)
   - **Protocol:** TCP
   - Clique em **ADD**.

   **Caminho dos Dados (Container Path: /data):**
   - Clique em **Add another Path, Port, Variable, Label or Device**.
   - **Config Type:** Path
   - **Name:** Appdata
   - **Host Path:** `/mnt/user/appdata/spl` (Recomendado)
   - **Container Path:** `/data` (NÃO ALTERAR)
   - **Access Mode:** Read/Write
   - Clique em **ADD**.

6. Clique em **APPLY**.

O Unraid irá baixar a imagem e iniciar o container. Assim que terminar, você poderá acessar o aplicativo clicando no ícone do container e selecionando **WebUI** ou acessando `http://SEU_IP_UNRAID:8080`.

## Atualização

Como a imagem está configurada com a tag `:latest`, para atualizar:
1. Vá na aba Docker.
2. Clique em "Check for Updates" (ou aguarde a verificação automática).
3. Se houver atualização, clique em "apply update" ao lado do container SPL.
