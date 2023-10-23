package parser

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

func Run(ctx context.Context, path string, api_prefix string) string {
	files := make(chan File)
	routes := make(chan Route)

	go get_files(ctx, path, api_prefix, files)

	go parse_routes(ctx, files, routes)

	return build_dts(ctx, routes)
	// fmt.Println(build_dts(routes))
}

// like app/v1/product/get.ts -> app/v1/product.get
func replae_postfix_for_base_module(path string) string {
	pattern := `(.+)/(get|delete|put|post)`

	re := regexp.MustCompile(pattern)

	replaced := re.ReplaceAllString(path, "$1.$2")

	return replaced
}

func remove_method_postfix(url string) string {
	pattern := `(.+).(get|delete|put|post)`

	re := regexp.MustCompile(pattern)

	replaced := re.ReplaceAllString(url, "$1")

	return replaced
}

func replace_dots_with_slashes(url string) string {
	return strings.ReplaceAll(url, ".", "/")
}

func get_route_from_path(path string) string {
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimSuffix(path, ".ts")
	path = replae_postfix_for_base_module(path)

	return path
}

func build_dts(ctx context.Context, routes chan Route) string {
	res := strings.Builder{}
	header := strings.Builder{}
	header.WriteString("// THIS IS GENERATED FILE\n// DO NOT EDIT\n\n")

	imports := make(map[string]string)

	res.WriteString("export default {\n")

	for r := range routes {
		for iden := range r.identifiers.s {
			path := r.imports.Get(iden)
			if path != "" {
				imports[path] = iden
			}
		}
		res.WriteString(fmt.Sprintf("  \"%s\": %s,\n", strings.Replace(get_route_from_path(r.path), "api/v1/", "", 1), r.to_string()))
	}

	res.WriteRune('}')

	for import_path, iden := range imports {
		header.WriteString(fmt.Sprintf("import { %s } from \"%s\"\n", iden, import_path))
	}

	return header.String() + "\n" + res.String()
}
