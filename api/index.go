package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Vango is the content of vango.json
type Vango struct {
	Packages  []Packages `json:"packages"`
	Generator Generator  `json:"generator"`
}

// About describes the project
type About struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description"`
}

// Packages is the list of projects.
type Packages struct {
	Private bool     `json:"private"`
	Name    string   `json:"name"`
	Host    string   `json:"host,omitempty"`
	Path    string   `json:"path"`
	Icon    string   `json:"icon,omitempty"`
	Git     string   `json:"git"`
	Sources []string `json:"sources"`
	About   About    `json:"about,omitempty"`
}

// Author is the author information.
type Author struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Generator is the vercel-vango information.
type Generator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
	Author  Author `json:"author"`
}

// DataPack is the content sent to templates.
type DataPack struct {
	Link      string
	Git       string
	GitHost   string
	Name      string
	Sources   string
	Private   bool
	About     About
	Generator Generator
}

// Handler is the Vercel handler.
func Handler(w http.ResponseWriter, r *http.Request) {
	jsn := GetAsset(r, "vango.json") // Get the vango.json file

	var vango Vango
	json.Unmarshal([]byte(jsn), &vango)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	name := regexp.MustCompile("[@/]").Split(strings.ToLower(strings.TrimPrefix(r.URL.Path, "/")), -1)[0] // Get the project name
	if len(name) >= 1 {
		for i := range vango.Packages {
			if strings.EqualFold(vango.Packages[i].Name, name) {
				pkg := vango.Packages[i]
				tmpl := map[bool]string{
					true:  "_assets/templates/forward.gohtml",
					false: "_assets/templates/project.gohtml",
				}[r.URL.Query().Get("go-get") == "1"]
				path := struct{ Path string }{Path: pkg.Path}

				var git bytes.Buffer
				template.Must(template.New("git").Parse(pkg.Git)).Execute(&git, path)

				var sources bytes.Buffer
				template.Must(template.New("sources").Parse(strings.Join(pkg.Sources, " "))).Execute(&sources, path)

				gh, err := url.Parse(git.String())
				if err != nil {
					log.Fatal(err)
				}

				template.Must(template.New("webpage").Parse(GetAsset(r, tmpl))).Execute(w, DataPack{
					Link: strings.Join([]string{r.Header.Get("x-forwarded-host"), name}, "/"),
					Git:  git.String(),
					GitHost: formatHost(map[bool]string{
						true:  gh.Host,
						false: pkg.Host,
					}[pkg.Host == ""]),
					Name:      pkg.Name,
					Sources:   sources.String(),
					Private:   pkg.Private,
					About:     pkg.About,
					Generator: vango.Generator,
				})
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404 - Page Not Found")
	} else {
		template.Must(template.New("webpage").Parse(GetAsset(r, "_assets/templates/index.gohtml"))).Execute(w, vango)
	}
}

// GetAsset fetches a static asset from a web server.
func GetAsset(r *http.Request, asset string) string {
	resp, err := http.Get(strings.Join([]string{
		r.Header.Get("x-forwarded-proto"),
		"://",
		r.Header.Get("X-Vercel-Deployment-Url"),
		"/",
		asset,
	}, ""))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(data)
}

// convert the domain to a (common) readable name
func formatHost(host string) string {
	switch h := strings.ToLower(host); h {
	case "github.com":
		return "GitHub"
	case "gitlab.com":
		return "GitLab"
	case "bitbucket.org":
		return "Bitbucket"
	default:
		return host
	}
}
