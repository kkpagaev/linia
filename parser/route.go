package parser

import (
	"fmt"
	"strings"
	"sync"
)

// zod schemas
type RouteSchema struct {
	body    string
	params  string
	headers string
	query   string
}

type void struct{}

var _ void

type Identifiers struct {
	sync.Mutex
	s map[string]void
}

func (i *Identifiers) Add(id string) {
	i.Lock()
	i.s[id] = void{}
	i.Unlock()
}

func (i *Identifiers) Contains(id string) bool {
	i.Lock()
	_, ok := i.s[id]
	i.Unlock()
	return ok
}

func (i *Identifiers) Remove(id string) {
	i.Lock()
	delete(i.s, id)
	i.Unlock()
}

func (i *Identifiers) Size() int {
	i.Lock()
	size := len(i.s)
	i.Unlock()
	return size
}

func (i *Identifiers) Clear() {
	i.Lock()
	i.s = make(map[string]void)
	i.Unlock()
}

func (i *Identifiers) Merge(other *Identifiers) {
	i.Lock()
	other.Lock()
	for k := range other.s {
		i.s[k] = void{}
	}
	other.Unlock()
	i.Unlock()
}

type Imports struct {
	sync.Mutex
	m map[string]string
}

func (i *Imports) Add(id string, path string) {
	i.Lock()
	i.m[id] = path
	i.Unlock()
}

func (i *Imports) Contains(id string) bool {
	i.Lock()
	_, ok := i.m[id]
	i.Unlock()
	return ok
}

func (i *Imports) Remove(id string) {
	i.Lock()
	delete(i.m, id)
	i.Unlock()
}

func (i *Imports) Get(id string) string {
	i.Lock()
	path := i.m[id]
	i.Unlock()
	return path
}

type Route struct {
	method      string
	contentType string
	is_admin    bool
	// role        string
	schema      RouteSchema
	path        string
	auth        bool
	identifiers *Identifiers
	imports     *Imports
}

func (r *Route) to_string() string {
	var s strings.Builder
	s.WriteRune('{')
	s.WriteRune('\n')

	s.WriteString(fmt.Sprintf("    method: \"%s\",\n", r.method))
	s.WriteString(fmt.Sprintf("    path: \"%s\",\n", r.path))
	s.WriteString(fmt.Sprintf("    url: \"%s\",\n", replace_dots_with_slashes(remove_method_postfix(get_route_from_path(r.path)))))
	s.WriteString(fmt.Sprintf("    auth: %t,\n", r.auth))
	s.WriteString(fmt.Sprintf("    isAdmin: %t,\n", r.is_admin))
	{
		var schema strings.Builder
		schema.WriteString("    schema: z.object({\n")

		if r.schema.body != "" {
			schema.WriteString(fmt.Sprintf("body: %s,\n", r.schema.body))
		}

		if r.schema.params != "" {
			schema.WriteString(fmt.Sprintf("params: %s,\n", r.schema.params))
		}

		if r.schema.headers != "" {
			schema.WriteString(fmt.Sprintf("headers: %s,\n", r.schema.headers))
		}

		if r.schema.query != "" {
			schema.WriteString(fmt.Sprintf("query: %s,\n", r.schema.query))
		}

		schema.WriteString("})\n")
		s.WriteString(schema.String())
	}

	s.WriteString("  ")
	s.WriteRune('}')

	return s.String()
}
