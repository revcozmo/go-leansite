package leansite

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/goforks/blackfriday"

	uio "github.com/metaleap/go-util/io"
)

func reloadTemplates() {
	fileNames := []string{filepath.Join(dir("templates"), "main.html")}
	uio.NewDirWalker(false, nil, func(_ *uio.DirWalker, fullPath string, _ os.FileInfo) bool {
		if !strings.HasSuffix(fullPath, string(filepath.Separator)+"main.html") {
			fileNames = append(fileNames, fullPath)
		}
		return true
	}).Walk(dir("templates"))
	var err error
	tmpl, err = template.ParseFiles(fileNames...)
	if err != nil {
		tmpl, err = template.New("error").Parse(fmt.Sprintf("ERROR loading templates: %+v", err))
	}
	return
}

func serveTemplatedContent(w http.ResponseWriter, r *http.Request) {
	urlPath := strings.Trim(r.URL.Path, "/")
	//	First handle static files (robots.txt / sitemap.xml / favicon.ico etc.) etc via 'static' folder
	if uio.FileExists(filepath.Join(dir("static"), urlPath)) {
		fileServer.ServeHTTP(w, r)
		return
	}

	var (
		err        error
		filePath   string
		fileData   []byte
		isMarkdown bool
	)
	if filePath = filepath.Join(dir("contents"), urlPath) + ".html"; !uio.FileExists(filePath) {
		if filePath = filepath.Join(dir("contents"), urlPath) + "/index.html"; !uio.FileExists(filePath) {
			isMarkdown = true
			if filePath = filepath.Join(dir("contents"), urlPath) + ".md"; !uio.FileExists(filePath) {
				filePath = filepath.Join(dir("contents"), urlPath) + "/index.md"
			}
		}
	}
	pc := NewPageContext(r, urlPath)
	if len(filePath) > 0 && uio.FileExists(filePath) {
		if fileData, err = ioutil.ReadFile(filePath); err == nil {
			var tmpl *template.Template
			if pos := bytes.Index(fileData, []byte("{{")); pos >= 0 {
				if bytes.Index(fileData, []byte("}}")) > pos {
					tmpl = pageTemplates[filePath]
					if tmpl == nil {
						var err error
						tmpl, err = template.ParseFiles(filePath)
						if err == nil {
							pageTemplates[filePath] = tmpl
						} else {
							tmpl, err = template.New("pterror").Parse(fmt.Sprintf("ERROR loading template %s:\t%+v", filePath, err))
						}
					}
				}
			}
			if tmpl != nil {
				var buf bytes.Buffer
				if err := tmpl.Execute(&buf, pc); err != nil {
					fileData = []byte(err.Error())
				} else {
					fileData = buf.Bytes()
				}
			}
			if isMarkdown {
				pc.HtmlContent = template.HTML(blackfriday.MarkdownCommon(fileData))
			} else {
				pc.HtmlContent = template.HTML(fileData)
			}
		}
	} else {
		pc.HtmlContent = "404 Not Found"
	}
	if err == nil {
		err = tmpl.Execute(w, pc)
	} else {
		w.Write([]byte(err.Error()))
	}
}
