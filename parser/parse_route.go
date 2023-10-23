package parser

import (
	"context"
	"fmt"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

const (
	GET     = "GET"
	POST    = "POST"
	PUT     = "PUT"
	PATCH   = "PATCH"
	DELETE  = "DELETE"
	OPTIONS = "OPTIONS"
)

func get_method_from_path(path string) (string, error) {
	if strings.HasSuffix(path, "get.ts") {
		return GET, nil
	}
	if strings.HasSuffix(path, "post.ts") {
		return POST, nil
	}
	if strings.HasSuffix(path, "put.ts") {
		return PUT, nil
	}
	if strings.HasSuffix(path, "patch.ts") {
		return PATCH, nil
	}
	if strings.HasSuffix(path, "delete.ts") {
		return DELETE, nil
	}
	if strings.HasSuffix(path, "options.ts") {
		return OPTIONS, nil
	}
	return "", fmt.Errorf("method not found")
}

func parse_routes(ctx context.Context, files chan File, routes chan Route) {
	var wg sync.WaitGroup
	for file := range files {
		wg.Add(1)
		go func(file File) {
			defer wg.Done()

			route, err := parse_route(ctx, file)

			if err != nil {
				panic(err)
			}

			select {
			case <-ctx.Done():
				return
			case routes <- route:
			}
		}(file)
	}
	wg.Wait()
	close(routes)
}

func parse_route(ctx context.Context, file File) (Route, error) {
	sourceCode := file.content

	route := Route{
		contentType: "application/json",
		is_admin:    false,
		auth:        false,
		schema:      RouteSchema{},
		path:        file.path,
		identifiers: &Identifiers{s: make(map[string]void)},
	}

	method, err := get_method_from_path(file.path)

	if err != nil {
		return route, err
	}

	route.method = method

	lang := typescript.GetLanguage()
	n, err := sitter.ParseCtx(ctx, sourceCode, lang)

	if err != nil {
		return route, err
	}

	imports, err := parse_imports(ctx, n, sourceCode, file.path)

	if err != nil {
		return route, err
	}

	route.imports = imports
	// fmt.Println(imports)

	err = parse_create_route(&route, n, sourceCode)

	if err != nil {
		return route, err
	}

	return route, nil
}

func resolve_import_relative_path(import_path string, path string) string {
	count := strings.Count(import_path, "../")

	if count == 0 {
		if strings.HasPrefix(import_path, "./") {
			return path + strings.TrimPrefix(import_path, "./")
		}

		return import_path
	}

	dirs := "./" + strings.Join(strings.Split(import_path, "/")[count:], "/")

	return dirs
}

func parse_imports(ctx context.Context, n *sitter.Node, sourceCode []byte, path string) (*Imports, error) {
	keyValuePattern := `
(import_statement
   (import_clause 
     (named_imports 
      (import_specifier 
          name: (identifier) @identifier
       )
      )
    )
  source: (string 
    (string_fragment) @path
           )
 ) 
`
	q, err := sitter.NewQuery([]byte(keyValuePattern), typescript.GetLanguage())

	if err != nil {
		return nil, err
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, n)

	imports := Imports{m: make(map[string]string)}

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		m = qc.FilterPredicates(m, sourceCode)
		identifiers := Identifiers{s: make(map[string]void)}
		var import_path string

		for _, c := range m.Captures {
			switch c.Node.Type() {
			case "identifier":
				identifiers.Add(c.Node.Content(sourceCode))
			case "string_fragment":
				import_path = resolve_import_relative_path(c.Node.Content(sourceCode), path)
			}
		}

		for iden := range identifiers.s {
			imports.Add(iden, import_path)
		}
	}

	return &imports, nil
}

func parse_create_route(route *Route, n *sitter.Node, sourceCode []byte) error {
	keyValuePattern := `
      (export_statement
        value: (call_expression
            function: (identifier) @_name (#eq? @_name "createRoute")
            arguments: (arguments
              (object (pair) @pair) 
               )
      ) 
    )
  `
	q, err := sitter.NewQuery([]byte(keyValuePattern), typescript.GetLanguage())

	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, n)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		m = qc.FilterPredicates(m, sourceCode)
		for _, c := range m.Captures {
			if c.Index == 0 {
				continue
			}

			parse_create_route_keys(c, route, c.Node, sourceCode)
		}
	}

	return nil
}

func parse_create_route_keys(c sitter.QueryCapture, route *Route, n *sitter.Node, sourceCode []byte) {
	key := c.Node.Child(0).Content(sourceCode)
	valueNode := c.Node.Child(2)
	value := valueNode.Content(sourceCode)

	switch key {
	case "auth":
		route.auth = value == "true"
	case "roles":
		parse_identiers(route, valueNode, sourceCode)
		route.is_admin = value == "[Role.Admin]"
	case "headers":
		parse_identiers(route, valueNode, sourceCode)
		route.schema.headers = value
	case "params":
		parse_identiers(route, valueNode, sourceCode)
		route.schema.params = value
	case "query":
		parse_identiers(route, valueNode, sourceCode)
		route.schema.query = value
	case "body":
		parse_identiers(route, valueNode, sourceCode)
		route.schema.body = value
	default:
		// fmt.Println(key)
	}
}

func parse_identiers(r *Route, n *sitter.Node, sourceCode []byte) {
	identifierPattern := `
         (identifier) @identifier
       `
	q, err := sitter.NewQuery([]byte(identifierPattern), typescript.GetLanguage())

	if err != nil {
		panic(err)
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(q, n)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		m = qc.FilterPredicates(m, sourceCode)

		for _, c := range m.Captures {
			r.identifiers.Add(c.Node.Content(sourceCode))
		}
	}
}
