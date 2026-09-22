package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Pass     string
	From     string
	To       string
	Subject  string
	Interval time.Duration
	Insecure bool
	Ext      string
	Modo     string
	Unico    bool
	Pasta    string
}

func buscarArquivos(pasta, extRaw string) []string {
	extRaw = strings.TrimSpace(extRaw)
	if extRaw == "" {
		extRaw = "xml"
	}
	parts := strings.Split(extRaw, ",")
	var exts []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(strings.TrimPrefix(p, ".")))
		if p != "" {
			exts = append(exts, p)
		}
	}

	entries, err := os.ReadDir(pasta)
	if err != nil {
		return nil
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		nome := e.Name()
		// ignora arquivos do proprio programa
		if strings.EqualFold(nome, "envio.log") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(nome), ".exe") {
			continue
		}
		// ignora go.mod etc se pedir *
		if strings.HasPrefix(nome, ".") {
			continue
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(nome), "."))
		for _, want := range exts {
			if want == "*" || want == "todos" || want == "all" || ext == want {
				files = append(files, filepath.Join(pasta, nome))
				break
			}
		}
	}
	return files
}

func isModoUnico(modo string, unicoFlag bool) bool {
	if unicoFlag {
		return true
	}
	m := strings.ToLower(strings.TrimSpace(modo))
	return m == "unico" || m == "único" || m == "agrupado" || m == "unificado" || m == "todos" || m == "lote" || m == "single" || m == "one"
}

func main() {
	var cfg Config
	flag.StringVar(&cfg.Host, "host", "", "SMTP host ex: smtp.gmail.com ou webmail.suaempresa.com.br")
	flag.IntVar(&cfg.Port, "port", 587, "SMTP porta ex: 465 ou 587")
	flag.StringVar(&cfg.User, "user", "", "Usuario SMTP")
	flag.StringVar(&cfg.Pass, "pass", "", "Senha / App Password")
	flag.StringVar(&cfg.From, "from", "", "De (opcional, usa -user se vazio)")
	flag.StringVar(&cfg.To, "to", "", "Para: destino@empresa.com")
	flag.StringVar(&cfg.Subject, "subject", "Envio XML - ", "Prefixo do assunto")
	flag.DurationVar(&cfg.Interval, "interval", 2*time.Second, "Intervalo entre envios no modo individual ex: 2s, 500ms")
	flag.BoolVar(&cfg.Insecure, "insecure", false, "Ignora validacao de certificado self-signed (Zimbra proprio)")

	// Novas flags
	flag.StringVar(&cfg.Ext, "ext", "xml", "Extensão a enviar. Ex: xml | pdf | xml,pdf,xlsx | *")
	flag.StringVar(&cfg.Ext, "extensao", "xml", "Alias para -ext")
	flag.StringVar(&cfg.Modo, "modo", "individual", "Modo: individual (1 por email) ou unico (todos no mesmo email)")
	flag.StringVar(&cfg.Modo, "mode", "individual", "Alias para -modo")
	flag.BoolVar(&cfg.Unico, "unico", false, "Atalho: envia todos os anexos em um unico email")
	flag.BoolVar(&cfg.Unico, "agrupado", false, "Alias para -unico")

	flag.StringVar(&cfg.Pasta, "pasta", "", "Pasta onde estão os arquivos. Ex: C:\\NFe ou /home/nfe")
	flag.StringVar(&cfg.Pasta, "dir", "", "Alias para -pasta")
	flag.StringVar(&cfg.Pasta, "folder", "", "Alias para -pasta")

	flag.Parse()

	// Fallback ENV
	if cfg.Host == "" {
		cfg.Host = os.Getenv("SMTP_HOST")
	}
	if cfg.User == "" {
		cfg.User = os.Getenv("SMTP_USER")
	}
	if cfg.Pass == "" {
		cfg.Pass = os.Getenv("SMTP_PASS")
	}
	if cfg.From == "" {
		cfg.From = os.Getenv("EMAIL_FROM")
	}
	if cfg.To == "" {
		cfg.To = os.Getenv("EMAIL_TO")
	}
	if cfg.From == "" {
		cfg.From = cfg.User
	}

	if cfg.Host == "" || cfg.User == "" || cfg.Pass == "" || cfg.To == "" {
		log.Fatal("Faltam configs. Use -host, -user, -pass, -to ou variaveis de ambiente SMTP_HOST, SMTP_USER, SMTP_PASS, EMAIL_TO")
	}

	// Define pasta usada
	var pastaUsada string
	if cfg.Pasta != "" {
		pastaUsada = cfg.Pasta
		if _, err := os.Stat(pastaUsada); os.IsNotExist(err) {
			log.Fatalf("Pasta não existe: %s", pastaUsada)
		}
	} else {
		// tenta pasta do exe, se vazio tenta wd (funciona no go run)
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		filesExe := buscarArquivos(exeDir, cfg.Ext)
		if len(filesExe) > 0 {
			pastaUsada = exeDir
		} else {
			wd, _ := os.Getwd()
			pastaUsada = wd
		}
	}

	files := buscarArquivos(pastaUsada, cfg.Ext)
	if len(files) == 0 {
		log.Printf("Nenhum arquivo .%s encontrado em %s", cfg.Ext, pastaUsada)
		return
	}

	// Log em arquivo + console
	logPath := filepath.Join(pastaUsada, "envio.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	fmt.Printf("Pasta: %s | Ext: %s | Modo: %s | Arquivos: %d\n", pastaUsada, cfg.Ext, cfg.Modo, len(files))
	if isModoUnico(cfg.Modo, cfg.Unico) {
		fmt.Println("Modo: UNICO - todos os anexos em 1 e-mail")
	} else {
		fmt.Printf("Modo: INDIVIDUAL - 1 e-mail por arquivo (intervalo %s)\n", cfg.Interval)
	}

	// Autenticação no início
	dialer := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Pass)
	if cfg.Port == 465 {
		dialer.SSL = true
	}
	dialer.TLSConfig = &tls.Config{
		InsecureSkipVerify: cfg.Insecure,
		ServerName:         cfg.Host,
	}

	fmt.Println("Testando autenticação SMTP...")
	s, err := dialer.Dial()
	if err != nil {
		log.Fatalf("Falha na autenticação SMTP: %v", err)
	}
	s.Close()
	fmt.Println("Autenticação OK!")

	enviadosDir := filepath.Join(pastaUsada, "enviados")
	os.MkdirAll(enviadosDir, 0755)

	// MODO ÚNICO
	if isModoUnico(cfg.Modo, cfg.Unico) {
		fmt.Printf("Enviando %d arquivos em UM ÚNICO e-mail...\n", len(files))
		m := gomail.NewMessage()
		m.SetHeader("From", cfg.From)
		m.SetHeader("To", cfg.To)
		m.SetHeader("Subject", fmt.Sprintf("%s%d arquivos (%s)", cfg.Subject, len(files), cfg.Ext))

		var lista string
		var totalSize int64
		for _, f := range files {
			info, _ := os.Stat(f)
			if info != nil {
				totalSize += info.Size()
			}
			lista += "- " + filepath.Base(f) + "\n"
			m.Attach(f)
		}

		// Aviso se passar de 20MB (limite comum Zimbra)
		if totalSize > 20*1024*1024 {
			fmt.Printf("AVISO: Total de anexos %.2f MB pode exceder limite do servidor\n", float64(totalSize)/1024/1024)
		}

		m.SetBody("text/plain", fmt.Sprintf("Segue em anexo %d arquivo(s) (%s):\n\n%s\nEnviado em %s\nPasta: %s", len(files), cfg.Ext, lista, time.Now().Format("02/01/2006 15:04:05"), pastaUsada))

		if err := dialer.DialAndSend(m); err != nil {
			log.Fatalf("ERRO no envio único: %v", err)
		}
		fmt.Println("OK - E-mail único enviado!")
		for _, f := range files {
			os.Rename(f, filepath.Join(enviadosDir, filepath.Base(f)))
		}
		log.Printf("OK UNICO - %d arquivos (%s) enviados para %s | Pasta: %s", len(files), cfg.Ext, cfg.To, pastaUsada)
		return
	}

	// MODO INDIVIDUAL
	sucessos, falhas := 0, 0
	for i, filePath := range files {
		fileName := filepath.Base(filePath)
		fmt.Printf("[%d/%d] Enviando %s... ", i+1, len(files), fileName)

		m := gomail.NewMessage()
		m.SetHeader("From", cfg.From)
		m.SetHeader("To", cfg.To)
		m.SetHeader("Subject", fmt.Sprintf("%s%s", cfg.Subject, fileName))
		m.SetBody("text/plain", fmt.Sprintf("Segue arquivo: %s\nEnviado em %s", fileName, time.Now().Format("02/01/2006 15:04:05")))
		m.Attach(filePath)

		if err := dialer.DialAndSend(m); err != nil {
			log.Printf("ERRO %s: %v", fileName, err)
			fmt.Println("ERRO")
			falhas++
		} else {
			fmt.Println("OK")
			log.Printf("OK - %s", fileName)
			os.Rename(filePath, filepath.Join(enviadosDir, fileName))
			sucessos++
		}
		if i < len(files)-1 {
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			time.Sleep(cfg.Interval + jitter)
		}
	}
	fmt.Printf("\nConcluído! Sucessos: %d | Falhas: %d\nLog: %s\nEnviados: %s\n", sucessos, falhas, logPath, enviadosDir)
}
