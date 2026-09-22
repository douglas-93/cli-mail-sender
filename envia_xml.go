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
}

func main() {
	var cfg Config
	flag.StringVar(&cfg.Host, "host", "", "SMTP host")
	flag.IntVar(&cfg.Port, "port", 587, "SMTP porta")
	flag.StringVar(&cfg.User, "user", "", "Usuario SMTP")
	flag.StringVar(&cfg.Pass, "pass", "", "Senha")
	flag.StringVar(&cfg.From, "from", "", "De")
	flag.StringVar(&cfg.To, "to", "", "Para")
	flag.StringVar(&cfg.Subject, "subject", "Envio XML - ", "Prefixo do assunto")
	flag.DurationVar(&cfg.Interval, "interval", 2*time.Second, "Intervalo entre envios")
	flag.BoolVar(&cfg.Insecure, "insecure", false, "Ignora validacao de certificado")
	flag.Parse()

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
		log.Fatal("Faltam configs. Use flags -host, -user, -pass, -to")
	}

	// 1. Acha a pasta - funciona tanto no go run quanto no .exe compilado
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	files, _ := filepath.Glob(filepath.Join(exeDir, "*.xml"))
	fUpper, _ := filepath.Glob(filepath.Join(exeDir, "*.XML"))
	files = append(files, fUpper...)

	pastaUsada := exeDir
	if len(files) == 0 {
		wd, _ := os.Getwd()
		f2, _ := filepath.Glob(filepath.Join(wd, "*.xml"))
		f2Upper, _ := filepath.Glob(filepath.Join(wd, "*.XML"))
		f2 = append(f2, f2Upper...)
		if len(f2) > 0 {
			files = f2
			pastaUsada = wd
		}
	}

	if len(files) == 0 {
		log.Println("Nenhum .xml encontrado")
		return
	}

	// 2. Log em arquivo + console
	logPath := filepath.Join(pastaUsada, "envio.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	defer logFile.Close()
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	fmt.Printf("Pasta usada: %s\n", pastaUsada)
	fmt.Printf("Encontrados %d XMLs\n", len(files))

	// 3. Autenticação no início
	dialer := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Pass)
	if cfg.Port == 465 {
		dialer.SSL = true
	}
	dialer.TLSConfig = &tls.Config{
		InsecureSkipVerify: cfg.Insecure,
		ServerName:         cfg.Host,
	}

	fmt.Println("Testando autenticação SMTP...")
	sender, err := dialer.Dial()
	if err != nil {
		log.Fatalf("Falha na autenticação SMTP: %v", err)
	}
	sender.Close()
	fmt.Println("Autenticação OK!")
	log.Printf("Autenticacao OK em %s:%d como %s", cfg.Host, cfg.Port, cfg.User)

	// 4. Pasta de enviados
	enviadosDir := filepath.Join(pastaUsada, "enviados")
	os.MkdirAll(enviadosDir, 0755)

	sucessos, falhas := 0, 0

	for i, filePath := range files {
		fileName := filepath.Base(filePath)
		fmt.Printf("[%d/%d] Enviando %s... ", i+1, len(files), fileName)

		m := gomail.NewMessage()
		m.SetHeader("From", cfg.From)
		m.SetHeader("To", cfg.To)
		m.SetHeader("Subject", fmt.Sprintf("%s%s", cfg.Subject, fileName))
		m.SetBody("text/plain", fmt.Sprintf("Segue XML: %s\nEnviado em %s", fileName, time.Now().Format("02/01/2006 15:04:05")))
		m.Attach(filePath)

		if err := dialer.DialAndSend(m); err != nil {
			log.Printf("ERRO %s: %v", fileName, err)
			fmt.Println("ERRO")
			falhas++
		} else {
			fmt.Println("OK")
			log.Printf("OK - %s enviado para %s", fileName, cfg.To)
			sucessos++

			// Move para enviados para não reenviar
			dest := filepath.Join(enviadosDir, fileName)
			if err := os.Rename(filePath, dest); err != nil {
				log.Printf("AVISO: Enviado mas falhou ao mover %s: %v", fileName, err)
			}
		}

		if i < len(files)-1 {
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			time.Sleep(cfg.Interval + jitter)
		}
	}

	fmt.Printf("\nConcluído! Sucessos: %d | Falhas: %d\nLog: %s\nEnviados: %s\n", sucessos, falhas, logPath, enviadosDir)
	log.Printf("Concluido - Sucessos: %d Falhas: %d", sucessos, falhas)
}
