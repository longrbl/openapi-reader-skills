package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

const exitCodeUsage = 2
const exitCodeFileErr = 3

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(exitCodeUsage)
	}

	cmd := os.Args[1]

	switch cmd {
	case "help", "-h", "--help":
		if len(os.Args) > 2 {
			printCommandHelp(os.Args[2])
		} else {
			printUsage()
		}
		return
	case "list", "info", "search", "tag", "endpoint", "schema", "uses-field":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "openapi: missing spec file path")
			fmt.Fprintf(os.Stderr, "Usage: openapi %s <spec> [options]\n", cmd)
			os.Exit(exitCodeUsage)
		}
	case "upsert-endpoint", "remove-endpoint", "upsert-schema":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "openapi: missing spec file path")
			fmt.Fprintf(os.Stderr, "Usage: openapi %s <spec> [options]\n", cmd)
			os.Exit(exitCodeUsage)
		}
	default:
		fmt.Fprintf(os.Stderr, "openapi: unknown command '%s'\n", cmd)
		printUsage()
		os.Exit(exitCodeUsage)
	}

	specFile := os.Args[2]
	rest := os.Args[3:]

	switch cmd {
	case "list":
		cmdList(specFile, rest)
	case "info":
		cmdInfo(specFile, rest)
	case "search":
		cmdSearch(specFile, rest)
	case "tag":
		cmdTag(specFile, rest)
	case "endpoint":
		cmdEndpoint(specFile, rest)
	case "schema":
		cmdSchema(specFile, rest)
	case "uses-field":
		cmdUsesField(specFile, rest)
	case "upsert-endpoint":
		cmdUpsertEndpoint(specFile, rest)
	case "remove-endpoint":
		cmdRemoveEndpoint(specFile, rest)
	case "upsert-schema":
		cmdUpsertSchema(specFile, rest)
	}
}

func printUsage() {
	w := os.Stderr
	fmt.Fprintf(w, "Usage: openapi <command> <spec_file> [options]\n\n")
	fmt.Fprintf(w, "Query commands:\n")
	fmt.Fprintf(w, "  list    <spec>                  List all endpoints\n")
	fmt.Fprintf(w, "  info    <spec>                  Show API info\n")
	fmt.Fprintf(w, "  search  <spec> <keyword>        Search endpoints by keyword\n")
	fmt.Fprintf(w, "  tag     <spec> <tag>            List endpoints by tag\n")
	fmt.Fprintf(w, "  endpoint <spec>                 Get endpoint details\n")
	fmt.Fprintf(w, "  schema  <spec> <name>           Get schema definition\n")
	fmt.Fprintf(w, "  uses-field <spec> <field>       Find endpoints using a field\n")
	fmt.Fprintf(w, "\nWrite commands:\n")
	fmt.Fprintf(w, "  upsert-endpoint  <spec>          Add/update an endpoint\n")
	fmt.Fprintf(w, "  remove-endpoint  <spec>          Remove an endpoint\n")
	fmt.Fprintf(w, "  upsert-schema    <spec>           Add/update a schema\n")
	fmt.Fprintf(w, "\nCommon options:\n")
	fmt.Fprintf(w, "  --limit N         Limit results (default: all)\n")
	fmt.Fprintf(w, "  --compact         Compact one-line-per-endpoint output\n")
	fmt.Fprintf(w, "  --depth N         Max schema recursion depth (default: 0=unlimited)\n")
	fmt.Fprintf(w, "  --output json     Output as JSON instead of text\n")
	fmt.Fprintf(w, "  --group-by-tag    Group endpoints by tag (list command)\n")
	fmt.Fprintf(w, "  --ascii           Use ASCII-only characters (no Unicode box drawing)\n")
	fmt.Fprintf(w, "\nFor per-command details: openapi help <command>\n")
}

func printCommandHelp(cmd string) {
	switch cmd {
	case "list":
		fmt.Fprintln(os.Stderr, "Usage: openapi list <spec> [options]")
		fmt.Fprintln(os.Stderr, "  Lists all API endpoints.")
		fmt.Fprintln(os.Stderr, "  --limit N         Show only N results")
		fmt.Fprintln(os.Stderr, "  --compact         One endpoint per line")
		fmt.Fprintln(os.Stderr, "  --output json     JSON output")
		fmt.Fprintln(os.Stderr, "  --group-by-tag    Group by tag")
	case "info":
		fmt.Fprintln(os.Stderr, "Usage: openapi info <spec> [--output json]")
		fmt.Fprintln(os.Stderr, "  Shows API title, version, and description.")
	case "search":
		fmt.Fprintln(os.Stderr, "Usage: openapi search <spec> <keyword> [options]")
		fmt.Fprintln(os.Stderr, "  Searches endpoints by keyword (matches path, summary, tags).")
		fmt.Fprintln(os.Stderr, "  --limit N, --compact, --output json")
	case "tag":
		fmt.Fprintln(os.Stderr, "Usage: openapi tag <spec> <tag> [options]")
		fmt.Fprintln(os.Stderr, "  Lists endpoints filtered by tag.")
		fmt.Fprintln(os.Stderr, "  --limit N, --compact, --output json")
	case "endpoint":
		fmt.Fprintln(os.Stderr, "Usage: openapi endpoint <spec> --path P --method M [options]")
		fmt.Fprintln(os.Stderr, "  Shows detailed endpoint information (parameters, request, responses).")
		fmt.Fprintln(os.Stderr, "  --path P           API path (required)")
		fmt.Fprintln(os.Stderr, "  --method M         HTTP method (required)")
		fmt.Fprintln(os.Stderr, "  --paths P1,P2      Batch query multiple paths")
		fmt.Fprintln(os.Stderr, "  --depth N          Max schema depth")
		fmt.Fprintln(os.Stderr, "  --params-only      Only show parameters")
		fmt.Fprintln(os.Stderr, "  --request-only     Only show request body")
		fmt.Fprintln(os.Stderr, "  --response-only    Only show responses")
		fmt.Fprintln(os.Stderr, "  --fields-only      Only show field names")
		fmt.Fprintln(os.Stderr, "  --output json      JSON output")
	case "schema":
		fmt.Fprintln(os.Stderr, "Usage: openapi schema <spec> <name> [--depth N] [--fields-only] [--output json]")
		fmt.Fprintln(os.Stderr, "  Shows a schema definition with all $ref resolved.")
	case "uses-field":
		fmt.Fprintln(os.Stderr, "Usage: openapi uses-field <spec> <field_name> [--compact] [--output json]")
		fmt.Fprintln(os.Stderr, "  Finds all endpoints that reference a given field in their schemas.")
	case "upsert-endpoint":
		fmt.Fprintln(os.Stderr, "Usage: openapi upsert-endpoint <spec> --path P --method M [options]")
		fmt.Fprintln(os.Stderr, "  Adds or updates an API endpoint.")
		fmt.Fprintln(os.Stderr, "  --file F           JSON file with full endpoint definition")
		fmt.Fprintln(os.Stderr, "  --summary S        Short description")
		fmt.Fprintln(os.Stderr, "  --description D    Detailed description")
		fmt.Fprintln(os.Stderr, "  --operation-id ID")
		fmt.Fprintln(os.Stderr, "  --tag-param T      Comma-separated tags")
		fmt.Fprintln(os.Stderr, "  --deprecated       Mark as deprecated")
		fmt.Fprintln(os.Stderr, "  --diff             Only show validation warnings")
	case "remove-endpoint":
		fmt.Fprintln(os.Stderr, "Usage: openapi remove-endpoint <spec> --path P [--method M]")
		fmt.Fprintln(os.Stderr, "  Removes an endpoint method or entire path.")
	case "upsert-schema":
		fmt.Fprintln(os.Stderr, "Usage: openapi upsert-schema <spec> --name N --file F | --json J")
		fmt.Fprintln(os.Stderr, "  Adds or updates a schema definition.")
		fmt.Fprintln(os.Stderr, "  --name N     Schema name")
		fmt.Fprintln(os.Stderr, "  --file F     JSON file with schema definition")
		fmt.Fprintln(os.Stderr, "  --json J     JSON string of schema definition")
		fmt.Fprintln(os.Stderr, "  --diff       Only show validation warnings")
	default:
		fmt.Fprintf(os.Stderr, "openapi: unknown command '%s'\n", cmd)
		printUsage()
	}
}

// ── Query commands ─────────────────────────────────────────────────

type queryFlags struct {
	limit      int
	compact    bool
	ascii      bool
	outputFmt  string
	groupByTag bool
}

func parseQueryFlags(fs *flag.FlagSet, args []string) queryFlags {
	var qf queryFlags
	fs.IntVar(&qf.limit, "limit", 0, "Limit results")
	fs.BoolVar(&qf.compact, "compact", false, "Compact output")
	fs.BoolVar(&qf.ascii, "ascii", false, "ASCII-only output")
	fs.StringVar(&qf.outputFmt, "output", "", `Output format ("json" for JSON)`)
	fs.BoolVar(&qf.groupByTag, "group-by-tag", false, "Group by tag")
	_ = fs.Parse(args)
	return qf
}

func isJSON(qf queryFlags) bool { return qf.outputFmt == "json" }

func largeSpecHint(n int) string {
	if n < 50 {
		return ""
	}
	return fmt.Sprintf("[hint] spec has %d endpoints. Use --limit, --search, or --tag to narrow output.\n", n)
}

func cmdList(specFile string, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	qf := parseQueryFlags(fs, args)

	api := mustLoad(specFile)

	if qf.groupByTag {
		groups := api.EndpointsGroupedByTag()
		if isJSON(qf) {
			fmt.Print(jsonOutput(groups))
		} else {
			fmt.Print(formatListGrouped(groups))
		}
		return
	}

	all := api.ListEndpoints()
	hint := largeSpecHint(len(all))
	if qf.limit > 0 && qf.limit < len(all) {
		all = all[:qf.limit]
	}
	if hint != "" && !isJSON(qf) {
		fmt.Fprint(os.Stderr, hint)
	}

	if isJSON(qf) {
		fmt.Print(jsonOutput(all))
	} else {
		fmt.Print(maybeASCII(formatList(all, "All Endpoints", qf.compact), qf.ascii))
	}
}

func cmdInfo(specFile string, args []string) {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	qf := parseQueryFlags(fs, args)

	api := mustLoad(specFile)
	info := api.Info()

	if isJSON(qf) {
		fmt.Print(jsonOutput(info))
	} else {
		fmt.Print(maybeASCII(formatInfo(info), qf.ascii))
	}
}

func cmdSearch(specFile string, args []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "Usage: openapi search <spec> <keyword> [options]")
		os.Exit(exitCodeUsage)
	}
	keyword := args[0]

	fs := flag.NewFlagSet("search", flag.ExitOnError)
	qf := parseQueryFlags(fs, args[1:])

	api := mustLoad(specFile)
	results := api.SearchEndpoints(keyword)
	if qf.limit > 0 && qf.limit < len(results) {
		results = results[:qf.limit]
	}

	if isJSON(qf) {
		fmt.Print(jsonOutput(results))
	} else {
		fmt.Print(maybeASCII(formatList(results, "Search: "+keyword, qf.compact), qf.ascii))
	}
}

func cmdTag(specFile string, args []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "Usage: openapi tag <spec> <tag> [options]")
		os.Exit(exitCodeUsage)
	}
	tag := args[0]

	fs := flag.NewFlagSet("tag", flag.ExitOnError)
	qf := parseQueryFlags(fs, args[1:])

	api := mustLoad(specFile)
	results := api.EndpointsByTag(tag)
	if qf.limit > 0 && qf.limit < len(results) {
		results = results[:qf.limit]
	}

	if isJSON(qf) {
		fmt.Print(jsonOutput(results))
	} else {
		fmt.Print(maybeASCII(formatList(results, "Tag: "+tag, qf.compact), qf.ascii))
	}
}

func cmdEndpoint(specFile string, args []string) {
	fs := flag.NewFlagSet("endpoint", flag.ExitOnError)
	path := fs.String("path", "", "API path")
	method := fs.String("method", "", "HTTP method")
	paths := fs.String("paths", "", "Comma-separated batch paths")
	depth := fs.Int("depth", 0, "Max schema depth")
	paramsOnly := fs.Bool("params-only", false, "Only parameters")
	requestOnly := fs.Bool("request-only", false, "Only request body")
	responseOnly := fs.Bool("response-only", false, "Only responses")
	fieldsOnly := fs.Bool("fields-only", false, "Only field names")
	ascii := fs.Bool("ascii", false, "ASCII-only output")
	outputFmt := fs.String("output", "", `Output format ("json" for JSON)`)
	_ = fs.Parse(args)

	api := mustLoad(specFile)
	jsonMode := *outputFmt == "json"

	if *paths != "" {
		if *method == "" {
			fmt.Fprintln(os.Stderr, "--method is required with --paths")
			os.Exit(exitCodeUsage)
		}
		for _, p := range strings.Split(*paths, ",") {
			p = strings.TrimSpace(p)
			ep := api.GetEndpoint(p, *method)
			if ep == nil {
				suggestPath(api, p, *method)
				continue
			}
			if jsonMode {
				fmt.Print(jsonOutput(ep))
			} else {
				fmt.Print(maybeASCII(api.formatEndpoint(ep, *depth, *paramsOnly, *requestOnly, *responseOnly, *fieldsOnly), *ascii))
			}
		}
		return
	}

	if *path == "" || *method == "" {
		fmt.Fprintln(os.Stderr, "--path and --method are required")
		os.Exit(exitCodeUsage)
	}

	ep := api.GetEndpoint(*path, *method)
	if ep == nil {
		suggestPath(api, *path, *method)
		return
	}

	if jsonMode {
		fmt.Print(jsonOutput(ep))
	} else {
		fmt.Print(maybeASCII(api.formatEndpoint(ep, *depth, *paramsOnly, *requestOnly, *responseOnly, *fieldsOnly), *ascii))
	}
}

func suggestPath(api *OpenAPI, path, method string) {
	fmt.Printf("Endpoint %s %s not found.\n", strings.ToUpper(method), path)
	// Find similar paths
	paths, _ := api.raw["paths"].(map[string]any)
	var suggestions []string
	for p := range paths {
		if strings.Contains(p, path) || strings.Contains(path, p) {
			suggestions = append(suggestions, p)
		}
	}
	if len(suggestions) > 0 {
		fmt.Printf("  Did you mean one of these?\n")
		for _, s := range suggestions {
			fmt.Printf("    %s\n", s)
		}
	}
}

func cmdSchema(specFile string, args []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "Usage: openapi schema <spec> <name> [--depth N] [--fields-only] [--output json]")
		os.Exit(exitCodeUsage)
	}
	name := args[0]

	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	depth := fs.Int("depth", 0, "Max schema depth")
	fieldsOnly := fs.Bool("fields-only", false, "Only field names")
	outputFmt := fs.String("output", "", `Output format ("json" for JSON)`)
	_ = fs.Parse(args[1:])

	api := mustLoad(specFile)
	schema := api.GetSchema(name)

	if *outputFmt == "json" {
		if schema == nil {
			suggestSchema(api, name)
		} else {
			fmt.Print(jsonOutput(schema))
		}
	} else {
		if schema == nil {
			suggestSchema(api, name)
		} else {
			fmt.Print(formatSchema(name, schema, *depth, *fieldsOnly))
		}
	}
}

func suggestSchema(api *OpenAPI, name string) {
	fmt.Printf("Schema '%s' not found.\n", name)
	// List available schemas
	schemas := getSchemata(api)
	if len(schemas) > 0 {
		fmt.Printf("  Available schemas: %s\n", strings.Join(schemas, ", "))
	} else {
		fmt.Printf("  No schemas defined in this spec.\n")
	}
}

func getSchemata(api *OpenAPI) []string {
	if api.isV2 {
		defs, _ := api.raw["definitions"].(map[string]any)
		return sortedKeys(defs)
	}
	components, _ := api.raw["components"].(map[string]any)
	if components == nil {
		return nil
	}
	schemas, _ := components["schemas"].(map[string]any)
	return sortedKeys(schemas)
}

func cmdUsesField(specFile string, args []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "Usage: openapi uses-field <spec> <field_name> [--compact] [--output json]")
		os.Exit(exitCodeUsage)
	}
	fieldName := args[0]

	fs := flag.NewFlagSet("uses-field", flag.ExitOnError)
	compact := fs.Bool("compact", false, "Compact output")
	outputFmt := fs.String("output", "", `Output format ("json" for JSON)`)
	_ = fs.Parse(args[1:])

	api := mustLoad(specFile)
	results := api.UsesField(fieldName)

	if *outputFmt == "json" {
		fmt.Print(jsonOutput(results))
	} else {
		fmt.Print(formatList(results, "Uses field: "+fieldName, *compact))
	}
}

// ── Write commands ─────────────────────────────────────────────────

func cmdUpsertEndpoint(specFile string, args []string) {
	if isURL(specFile) {
		fmt.Fprintln(os.Stderr, "error: write commands require a local file, not a URL")
		os.Exit(exitCodeUsage)
	}
	fs := flag.NewFlagSet("upsert-endpoint", flag.ExitOnError)
	path := fs.String("path", "", "API path")
	method := fs.String("method", "", "HTTP method")
	file := fs.String("file", "", "JSON file with endpoint definition")
	summary := fs.String("summary", "", "Short description")
	description := fs.String("description", "", "Detailed description")
	tagParam := fs.String("tag-param", "", "Tags (comma-separated)")
	operationID := fs.String("operation-id", "", "Operation ID")
	deprecated := fs.Bool("deprecated", false, "Mark deprecated")
	paramsJSON := fs.String("params-json", "", "Parameters JSON string")
	requestBodyJSON := fs.String("request-body", "", "Request body JSON string")
	responsesJSON := fs.String("responses", "", "Responses JSON string")
	diff := fs.Bool("diff", false, "Only show validation warnings")
	_ = fs.Parse(args)

	if *path == "" || *method == "" {
		fmt.Fprintln(os.Stderr, "--path and --method are required")
		os.Exit(exitCodeUsage)
	}

	api := mustLoadForWrite(specFile)

	var operation map[string]any
	if *file != "" {
		op, err := loadJSONFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading endpoint file: %v\nCheck file path and JSON validity.\n", err)
			os.Exit(exitCodeFileErr)
		}
		operation = op
	} else {
		operation = make(map[string]any)
		if *tagParam != "" {
			tags := make([]any, 0)
			for _, t := range strings.Split(*tagParam, ",") {
				tags = append(tags, strings.TrimSpace(t))
			}
			operation["tags"] = tags
		}
		if *summary != "" {
			operation["summary"] = *summary
		}
		if *description != "" {
			operation["description"] = *description
		}
		if *operationID != "" {
			operation["operationId"] = *operationID
		}
		if *deprecated {
			operation["deprecated"] = true
		}
		if *paramsJSON != "" {
			var params []any
			if err := json.Unmarshal([]byte(*paramsJSON), &params); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing --params-json: %v\nSupply a valid JSON array.\n", err)
			} else {
				operation["parameters"] = params
			}
		}
		if *requestBodyJSON != "" {
			var rb any
			if err := json.Unmarshal([]byte(*requestBodyJSON), &rb); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing --request-body: %v\nSupply a valid JSON object.\n", err)
			} else {
				operation["requestBody"] = rb
			}
		}
		if *responsesJSON != "" {
			var resp any
			if err := json.Unmarshal([]byte(*responsesJSON), &resp); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing --responses: %v\nSupply a valid JSON object.\n", err)
			} else {
				operation["responses"] = resp
			}
		} else {
			operation["responses"] = map[string]any{"200": map[string]any{"description": "Success"}}
		}
	}

	result, warnings := UpsertEndpoint(api, *path, *method, operation)
	if err := SaveSpec(api, specFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving spec: %v\n", err)
		os.Exit(exitCodeFileErr)
	}

	title := api.Info().Title
	if title == "" {
		title = specFile
	}
	fmt.Printf("# %s\n", title)
	fmt.Println(result)
	if *diff {
		for _, w := range warnings {
			if strings.HasPrefix(w, "    [!]") {
				fmt.Println(w)
			}
		}
	} else if len(warnings) > 0 {
		fmt.Println(strings.Join(warnings, "\n"))
	}
}

func cmdRemoveEndpoint(specFile string, args []string) {
	if isURL(specFile) {
		fmt.Fprintln(os.Stderr, "error: write commands require a local file, not a URL")
		os.Exit(exitCodeUsage)
	}
	fs := flag.NewFlagSet("remove-endpoint", flag.ExitOnError)
	path := fs.String("path", "", "API path")
	method := fs.String("method", "", "HTTP method (omit to remove entire path)")
	_ = fs.Parse(args)

	if *path == "" {
		fmt.Fprintln(os.Stderr, "--path is required")
		os.Exit(exitCodeUsage)
	}

	api := mustLoadForWrite(specFile)
	result := RemoveEndpoint(api, *path, *method)
	if err := SaveSpec(api, specFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving spec: %v\n", err)
		os.Exit(exitCodeFileErr)
	}

	title := api.Info().Title
	if title == "" {
		title = specFile
	}
	fmt.Printf("# %s\n", title)
	fmt.Println(result)
}

func cmdUpsertSchema(specFile string, args []string) {
	if isURL(specFile) {
		fmt.Fprintln(os.Stderr, "error: write commands require a local file, not a URL")
		os.Exit(exitCodeUsage)
	}
	fs := flag.NewFlagSet("upsert-schema", flag.ExitOnError)
	name := fs.String("name", "", "Schema name")
	schemaFile := fs.String("file", "", "JSON file with schema definition")
	schemaJSON := fs.String("json", "", "JSON string of schema definition")
	diff := fs.Bool("diff", false, "Only show validation warnings")
	_ = fs.Parse(args)

	if *name == "" {
		fmt.Fprintln(os.Stderr, "--name is required")
		os.Exit(exitCodeUsage)
	}

	var schemaData any
	if *schemaFile != "" {
		data, err := loadJSONFile(*schemaFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading schema file: %v\nCheck file path and JSON validity.\n", err)
			os.Exit(exitCodeFileErr)
		}
		schemaData = data
	} else if *schemaJSON != "" {
		if err := json.Unmarshal([]byte(*schemaJSON), &schemaData); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing --json: %v\nSupply a valid JSON object.\n", err)
			os.Exit(exitCodeUsage)
		}
	} else {
		fmt.Fprintln(os.Stderr, "--file or --json is required")
		os.Exit(exitCodeUsage)
	}

	api := mustLoadForWrite(specFile)
	result, warnings := UpsertSchema(api, *name, schemaData)
	if err := SaveSpec(api, specFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving spec: %v\n", err)
		os.Exit(exitCodeFileErr)
	}

	title := api.Info().Title
	if title == "" {
		title = specFile
	}
	fmt.Printf("# %s\n", title)
	fmt.Println(result)
	if *diff {
		for _, w := range warnings {
			if strings.HasPrefix(w, "    [!]") {
				fmt.Println(w)
			}
		}
	} else if len(warnings) > 0 {
		fmt.Println(strings.Join(warnings, "\n"))
	}
}

// ── Helpers ────────────────────────────────────────────────────────

func mustLoad(path string) *OpenAPI {
	api, err := LoadSpec(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading spec: %v\n", err)
		os.Exit(exitCodeFileErr)
	}
	return api
}

func mustLoadForWrite(path string) *OpenAPI {
	if isURL(path) {
		fmt.Fprintln(os.Stderr, "error: write commands require a local file, not a URL")
		os.Exit(exitCodeUsage)
	}
	api, err := LoadSpec(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading spec: %v\n", err)
		os.Exit(exitCodeFileErr)
	}
	return api
}

func loadJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// maybeASCII replaces Unicode box-drawing characters with ASCII equivalents
// when ascii mode is enabled.
func maybeASCII(text string, ascii bool) string {
	if !ascii {
		return text
	}
	r := strings.NewReplacer(
		"\u2500", "-",  // ─ → -
		"\u2502", "|",  // │ → |
		"\u250c", "+",  // ┌ → +
		"\u2510", "+",  // ┐ → +
		"\u2514", "+",  // └ → +
		"\u2518", "+",  // ┘ → +
		"\u251c", "+",  // ├ → +
		"\u2524", "+",  // ┤ → +
		"\u252c", "+",  // ┬ → +
		"\u2534", "+",  // ┴ → +
		"\u253c", "+",  // ┼ → +
		"\u2550", "=",  // ═ → =
	)
	return r.Replace(text)
}
