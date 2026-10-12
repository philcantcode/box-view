package web

import (
	"encoding/json"
	"io"
	"strings"

	model "github.com/philcantcode/box-view"
)

// ExportHTML writes a self-contained, offline nested-box view. No backend is required.
func ExportHTML(w io.Writer, p model.Projection) error {
	css, err := files.ReadFile("style.css")
	if err != nil {
		return err
	}
	js, err := files.ReadFile("box-view.js")
	if err != nil {
		return err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	script := strings.ReplaceAll(string(js), "export ", "")
	_, err = io.WriteString(w, "<!doctype html><html lang=\"en\"><meta charset=\"utf-8\"><title>Box View</title><style>"+string(css)+"</style><body><h1>Box View</h1><box-view></box-view><pre id=\"inspector\"></pre><script type=\"module\">"+script+"\nconst p="+string(data)+";const viewer=document.querySelector('box-view');viewer.projection=p;viewer.addEventListener('select-edge',e=>{document.getElementById('inspector').textContent=JSON.stringify(e.detail,null,2)});</script></body></html>")
	return err
}
