package structexplorer

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
)

//go:embed index_tmpl.html
var indexHTML string

func (s *service) init() {
	tmpl := template.New("index")
	tmpl = tmpl.Funcs(template.FuncMap{
		"fieldvalue": func(f fieldEntry) string {
			return printString(f.Value())
		},
		"includeField": func(f fieldEntry, s string) bool {
			return s != "nil" || f.hideNil
		},
	})
	tmpl, err := tmpl.Parse(indexHTML)
	if err != nil {
		slog.Error("failed to parse template", "err", err)
	}
	s.indexTemplate = tmpl
}

type Options struct {
	HTTPPort int
	ServeMux *http.ServeMux
}

func (o *Options) httpPort() int {
	if o.HTTPPort == 0 {
		return 5656
	}
	return o.HTTPPort
}

func (o *Options) serveMux() *http.ServeMux {
	if o.ServeMux == nil {
		return http.DefaultServeMux
	}
	return o.ServeMux
}

type service struct {
	explorer      *explorer
	indexTemplate *template.Template
}

func NewService(labelValuePairs ...any) *service {
	return &service{explorer: newExplorerOnAll(labelValuePairs...)}
}

func (s *service) Start(opts ...Options) {
	if len(opts) > 0 {
		s.explorer.options = &opts[0]
	}
	s.init()
	port := s.explorer.options.httpPort()
	serveMux := s.explorer.options.serveMux()
	slog.Info(fmt.Sprintf("starting go struct explorer at http://localhost:%d on %v", port, s.explorer.rootKeys()))
	serveMux.HandleFunc("/", s.serveIndex)
	serveMux.HandleFunc("/instructions", s.serveInstructions)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		slog.Error("failed to start server", "err", err)
	}
}

func (s *service) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	if err := s.indexTemplate.Execute(w, s.explorer.buildIndexData()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type uiInstruction struct {
	Row        int      `json:"row"`
	Column     int      `json:"column"`
	Selections []string `json:"selections"`
	Action     string   `json:"action"`
}

func (s *service) serveInstructions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cmd := uiInstruction{}
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Debug("instruction", "row", cmd.Row, "column", cmd.Column, "selections", cmd.Selections, "action", cmd.Action)

	fromAccess := s.explorer.objectAt(cmd.Row, cmd.Column)
	toRow := cmd.Row
	toColumn := cmd.Column
	switch cmd.Action {
	case "down":
		toRow++
	case "right":
		toColumn++
	case "remove":
		s.explorer.removeObjectAt(cmd.Row, cmd.Column)
		return
	case "toggleNils":
		s.explorer.updateObjectAt(cmd.Row, cmd.Column, func(access objectAccess) objectAccess {
			access.hideNils = !access.hideNils
			return access
		})
		return
	default:
		slog.Warn("invalid direction", "action", cmd.Action)
		http.Error(w, "invalid action", http.StatusBadRequest)
		return
	}
	for _, each := range cmd.Selections {
		newPath := append(fromAccess.path, each)
		oa := objectAccess{
			root:  fromAccess.root,
			path:  newPath,
			label: strings.Join(newPath, "."),
		}
		v := oa.Value()
		if !canExplore(v) {
			slog.Warn("cannot explore this", "value", v)
			continue
		}
		oa.typeName = fmt.Sprintf("%T", v)
		s.explorer.objectAtPut(toRow, toColumn, oa)
	}
}
