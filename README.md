# CLI Mail Sender - Envio de Arquivos por E-mail

Enviador automático de arquivos em Go. Foi feito para enviar XMLs de NFe, mas agora funciona com qualquer extensão. Autentica uma única vez no início e respeita intervalo anti-spam.

### Funcionalidades

- **Pasta flexível**: por padrão usa a pasta do `.exe`, mas aceita `-pasta` para qualquer diretório
- **Filtro por extensão**: `-ext=xml` (padrão), `-ext=pdf`, `-ext=xml,pdf,xlsx` ou `-ext=*` para tudo
- **2 modos de envio**:
  - `individual` (padrão): 1 arquivo = 1 e-mail
  - `unico`: todos os arquivos em um único e-mail
- **Anti-bloqueio**: intervalo de 2s + jitter aleatório de 0-1s entre envios (configurável)
- **Zimbra / SSL**: suporte a porta 465 (SSL implícito) e 587 (STARTTLS) + flag `-insecure` para certificado self-signed
- **Organização automática**: cria pasta `enviados/` e move os arquivos enviados + gera `envio.log`

---

### Compilação

```bash
git clone https://github.com/douglas-93/cli-mail-sender.git
cd cli-mail-sender
go mod tidy
go build -o cli-mail-sender.exe
# Linux/Mac
go build -o cli-mail-sender
```

---

### Configuração

Pode passar por flags ou variáveis de ambiente.

**Flags principais:**
| Flag | Descrição | Exemplo |
|---|---|---|
| `-host` | Host SMTP | `webmail.suaempresa.com.br` |
| `-port` | Porta 465 ou 587 | `465` |
| `-user` | Usuário SMTP (e-mail completo) | `noreply@suaempresa.com.br` |
| `-pass` | Senha / App Password | `SuaSenha` |
| `-to` | Destinatário | `destino@cliente.com` |
| `-from` | Remetente (opcional) | `nfe@suaempresa.com.br` |
| `-pasta` | Pasta dos arquivos | `C:\NFe\2024` |
| `-ext` | Extensão | `xml` , `pdf` , `xml,pdf` , `*` |
| `-modo` | `individual` ou `unico` | `individual` |
| `-unico` | Atalho para modo único | `true` |
| `-interval` | Intervalo modo individual | `2s` , `500ms` , `5s` |
| `-insecure` | Ignora certificado self-signed | `true` |

**Via ENV (recomendado pra não expor senha no histórico):**
```powershell
$env:SMTP_HOST="webmail.suaempresa.com.br"
$env:SMTP_USER="noreply@suaempresa.com.br"
$env:SMTP_PASS="SuaSenha"
$env:EMAIL_TO="destino@cliente.com"
.\cli-mail-sender.exe -port=465 -ext=xml
```

---

### Exemplos de Uso

**1. Caso original - XMLs na mesma pasta do .exe, 1 por e-mail:**
```powershell
.\cli-mail-sender.exe -host="webmail.suaempresa.com.br" -port=465 -user="noreply@suaempresa.com.br" -pass="..." -to="cliente@suaempresa.com.br"
```

**2. Definir pasta diferente (não precisa deixar o .exe junto):**
```powershell
.\cli-mail-sender.exe -pasta="C:\Users\cliente\OneDrive\NFe" -ext=xml -to="destino@empresa.com" -host="..." -port=465 -user="..." -pass="..."
# Alias: -dir ou -folder funcionam igual
```

**3. Enviar apenas PDFs:**
```powershell
.\cli-mail-sender.exe -pasta="D:\Retornos" -ext=pdf -to="financeiro@empresa.com"
```

**4. Enviar XML + PDF + XLSX no mesmo dia:**
```powershell
.\cli-mail-sender.exe -pasta="C:\Retornos\21-09" -ext="xml,pdf,xlsx" -to="cliente@empresa.com"
```

**5. NOVO - Todos os arquivos em UM ÚNICO e-mail com 6 anexos:**
```powershell
# 6 XMLs viram 1 e-mail com 6 anexos
.\cli-mail-sender.exe -pasta="C:\NFe" -ext=xml -unico -to="contabilidade@empresa.com"

# ou
.\cli-mail-sender.exe -modo=unico -ext="xml,pdf" -pasta="C:\Pacote"
```

**6. Enviar tudo que estiver na pasta, independente da extensão:**
```powershell
.\cli-mail-sender.exe -ext="*" -unico -to="backup@empresa.com"
```

**7. Zimbra com certificado self-signed:**
```powershell
.\cli-mail-sender.exe -host="webmail.suaempresa.com.br" -port=465 -user="..." -pass="..." -to="..." -insecure=true -pasta="C:\NFe"
# Se for 587:
.\cli-mail-sender.exe -host="webmail.suaempresa.com.br" -port=587 -insecure=true -to="..."
```

**8. Gmail (precisa de Senha de App):**
```powershell
.\cli-mail-sender.exe -host="smtp.gmail.com" -port=587 -user="seu@gmail.com" -pass="SENHA_DE_APP" -to="destino@gmail.com" -ext=xml -pasta="C:\NFe"
```

---

### Estrutura após envio

```
C:\NFe\
  ├─ 3126091019...-nfe.xml  (antes)
  ├─ cli-mail-sender.exe
  ├─ envio.log              (criado automaticamente)
  └─ enviados\              (criado automaticamente)
       └─ 3126091019...-nfe.xml (depois de enviado)
```

- `envio.log`: histórico com data/hora, nome do arquivo e status
- `enviados/`: evita reenvio

### Intervalo - 1 segundo é suficiente?

- **< 50 arquivos em servidor próprio (Zimbra local)**: 1s OK
- **50-300 arquivos / Gmail / Outlook / Zimbra hospedado**: use 2-3s + jitter (padrão do programa). Padrão fixo de 1s é detectado como bot.
- **> 300 arquivos**: use 5s ou envie em modo `unico`.

O programa já adiciona `+ 0 a 1000ms` aleatórios em cada envio.

### Troubleshooting Zimbra

- `535 authentication failed`: usuário bloqueado, senha expirada ou precisa trocar no primeiro acesso. Teste login no webmail.
- `x509: certificate signed by unknown authority`: use `-insecure=true` se for Zimbra com certificado próprio.
- Conta `noreply`: algumas instalações bloqueiam relay para contas de sistema. Libere no admin: `Permitir envio SMTP`.

### Segurança

- Não deixe senha no comando. Use ENV ou arquivo `.env` e apague histórico do PowerShell: `Clear-History`
- Troque senhas que foram expostas em prints/logs
- Limite de anexo Zimbra padrão é 25MB. O programa avisa se passar de 20MB no modo único.

---
Feito em Go com `gopkg.in/gomail.v2`
