package main

import (
	"bufio"
	"embed"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed templates/*
var content embed.FS

var arquivoCfg = "centraisColeta.cfg"

type Central struct {
	Nome              string
	Caminho           string
	UltimaAtualizacao string
	Status            string
}

func main() {

	http.HandleFunc("/coleta", handler)

	http.ListenAndServe(":8080", nil)
}

func carregarCentrais(arquivoCfg string) ([]Central, error) {
	arq, err := os.Open(arquivoCfg)
	if err != nil {
		return nil, err
	}
	defer arq.Close()

	var centrais []Central
	sc := bufio.NewScanner(arq)

	for sc.Scan() {
		linha := strings.TrimSpace(sc.Text())
		if linha == "" || strings.HasPrefix(linha, "#") {
			continue
		}
		partes := strings.Split(linha, "|")
		if len(partes) < 2 {
			continue
		}
		centrais = append(centrais, Central{
			Nome:    strings.TrimSpace(partes[0]),
			Caminho: strings.TrimSpace(partes[1]),
		})
	}
	return centrais, nil
}

func atualizarStatus(c *Central) {
	arquivos, err := filepath.Glob(filepath.Join(c.Caminho, "*"))
	if err != nil || len(arquivos) == 0 {
		c.Status = "nok"
		return
	}

	var maisRecente time.Time

	for _, arq := range arquivos {
		info, err := os.Stat(arq)
		if err != nil || info.IsDir() {
			continue
		}

		if info.ModTime().After(maisRecente) {
			maisRecente = info.ModTime()
		}
	}

	if !maisRecente.IsZero() {
		c.UltimaAtualizacao = maisRecente.Format("02/01 15:04:05")
	} else {
		c.UltimaAtualizacao = "Sem arquivo"
	}

	if time.Since(maisRecente) <= time.Hour {
		c.Status = "ok"
	} else {
		c.Status = "nok"
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFS(content, "templates/index.html"))
	Centrais, err := carregarCentrais(arquivoCfg)
	if err != nil {
		panic(err)
	}

	for i := range Centrais {
		atualizarStatus(&Centrais[i])
	}

	tmpl.Execute(w, Centrais)
}
