package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"
)

// ── Types ──────────────────────────────────────────────────────────

type OpenAPI struct {
	raw       map[string]any
	isV2      bool
	refCache  map[string]any
	resolving map[string]bool
}

type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type EndpointSummary struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Summary     string   `json:"summary,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	OperationID string   `json:"operationId,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty"`
}

type ParamDetail struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Required    bool   `json:"required"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Default     any    `json:"default,omitempty"`
	Enum        []any  `json:"enum,omitempty"`
}

type BodyDetail struct {
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	MediaType   string `json:"mediaType"`
	Schema      any    `json:"schema,omitempty"`
}

type ResponseDetail struct {
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
	MediaType   string `json:"mediaType"`
	Schema      any    `json:"schema,omitempty"`
}

type EndpointDetail struct {
	Method      string           `json:"method"`
	Path        string           `json:"path"`
	Summary     string           `json:"summary,omitempty"`
	Description string           `json:"description,omitempty"`
	OperationID string           `json:"operationId,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Deprecated  bool             `json:"deprecated,omitempty"`
	Parameters  []ParamDetail    `json:"parameters,omitempty"`
	RequestBody *BodyDetail      `json:"requestBody,omitempty"`
	Responses   []ResponseDetail `json:"responses,omitempty"`
}

var httpMethods = []string{"get", "post", "put", "patch", "delete", "options", "head"}

// ── Loading ────────────────────────────────────────────────────────

func LoadSpec(filepathArg string) (*OpenAPI, error) {
	fp := fixMSYS2SpecPath(filepathArg)

	content, err := readFileAuto(fp)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("JSON parse error: %v\n(Is this a valid JSON file?)", err)
	}

	// Validate basic OpenAPI structure
	openapiVer, hasV3 := raw["openapi"].(string)
	swaggerVer, hasV2 := raw["swagger"].(string)
	if !hasV3 && !hasV2 {
		return nil, fmt.Errorf("not an OpenAPI spec file: missing 'openapi' (3.x) or 'swagger' (2.0) field")
	}
	_ = openapiVer
	_ = swaggerVer

	o := &OpenAPI{
		raw:       raw,
		isV2:      hasV2,
		refCache:  make(map[string]any),
		resolving: make(map[string]bool),
	}
	return o, nil
}

func readFileAuto(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if utf8.Valid(data) {
		if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
			return string(data[3:]), nil
		}
		return string(data), nil
	}

	// Try UTF-8 with BOM stripping
	for _, bom := range [][]byte{
		{0xEF, 0xBB, 0xBF},
		{0xFF, 0xFE},
		{0xFE, 0xFF},
	} {
		if len(data) >= len(bom) && equalBytes(data[:len(bom)], bom) {
			trimmed := data[len(bom):]
			if utf8.Valid(trimmed) {
				return string(trimmed), nil
			}
		}
	}

	// File is not UTF-8 — give a clear error
	_, fname := splitPath(path)
	return "", fmt.Errorf("file '%s' is not UTF-8 encoded.\nTip: save the file as UTF-8 (in VS Code: File → Save with Encoding → UTF-8)", fname)
}

func splitPath(s string) (string, string) {
	idx := strings.LastIndexAny(s, "/\\")
	if idx < 0 {
		return "", s
	}
	return s[:idx], s[idx+1:]
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func fixMSYS2SpecPath(raw string) string {
	if runtime.GOOS != "windows" {
		return raw
	}
	if !strings.HasPrefix(raw, "/") {
		return raw
	}
	// /c/Users/name/file.json → C:/Users/name/file.json
	if len(raw) > 3 && raw[0] == '/' && raw[2] == '/' && isLetter(raw[1]) {
		trial := strings.ToUpper(raw[1:2]) + ":" + raw[2:]
		trial = strings.ReplaceAll(trial, "/", "\\")
		if _, err := os.Stat(trial); err == nil {
			return trial
		}
	}
	// Strip leading slash for CWD-relative from Git Bash
	trial := raw[1:]
	if _, err := os.Stat(trial); err == nil {
		return trial
	}
	return raw
}

func fixMSYS2APIPath(raw string, specPaths []string) string {
	if runtime.GOOS != "windows" {
		return raw
	}
	for _, sp := range specPaths {
		if raw == sp {
			return raw
		}
	}
	normal := strings.ReplaceAll(raw, "\\", "/")
	for _, sp := range specPaths {
		if strings.HasSuffix(normal, sp) {
			return sp
		}
	}
	return raw
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// ── Ref Resolution ─────────────────────────────────────────────────

func (o *OpenAPI) resolveRef(ref string) any {
	if ref == "" {
		return nil
	}
	if cached, ok := o.refCache[ref]; ok {
		return cached
	}
	if o.resolving[ref] {
		return map[string]any{"type": "object", "description": "[Circular: " + ref + "]"}
	}
	if !strings.HasPrefix(ref, "#/") {
		return map[string]any{"type": "object", "description": "[External: " + ref + "]"}
	}

	o.resolving[ref] = true
	defer delete(o.resolving, ref)

	parts := splitRefPath(ref[2:])
	node := any(o.raw)
	for _, part := range parts {
		m, ok := node.(map[string]any)
		if !ok {
			o.refCache[ref] = nil
			return nil
		}
		child, exists := m[part]
		if !exists {
			o.refCache[ref] = nil
			return nil
		}
		node = child
	}

	// Return the raw node — no deep copy. Format functions resolve nested $ref on access.
	o.refCache[ref] = node
	return node
}

func splitRefPath(s string) []string {
	parts := strings.Split(s, "/")
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(p, "~1", "/"), "~0", "~")
	}
	return parts
}

func (o *OpenAPI) resolveSchema(node any) any {
	if node == nil {
		return nil
	}
	m, ok := node.(map[string]any)
	if !ok {
		return node
	}
	if ref, ok := m["$ref"].(string); ok {
		return o.resolveRef(ref)
	}
	return node
}

func (o *OpenAPI) schemaRefPath(name string) string {
	if o.isV2 {
		return "#/definitions/" + name
	}
	return "#/components/schemas/" + name
}

// ── Query Methods ──────────────────────────────────────────────────

func (o *OpenAPI) Info() Info {
	info, _ := o.raw["info"].(map[string]any)
	if info == nil {
		return Info{}
	}
	return Info{
		Title:       str(info["title"]),
		Version:     str(info["version"]),
		Description: str(info["description"]),
	}
}

func (o *OpenAPI) ListEndpoints() []EndpointSummary {
	paths, _ := o.raw["paths"].(map[string]any)
	var results []EndpointSummary
	for _, path := range sortedKeys(paths) {
		methods, _ := paths[path].(map[string]any)
		for _, m := range httpMethods {
			op, ok := methods[m].(map[string]any)
			if !ok {
				continue
			}
			results = append(results, EndpointSummary{
				Method:      strings.ToUpper(m),
				Path:        path,
				Summary:     str(op["summary"]),
				Tags:        toStringSlice(op["tags"]),
				OperationID: str(op["operationId"]),
				Deprecated:  boolVal(op["deprecated"]),
			})
		}
	}
	return results
}

func (o *OpenAPI) SearchEndpoints(kw string) []EndpointSummary {
	kw = strings.ToLower(kw)
	var results []EndpointSummary
	all := o.ListEndpoints()
	for _, ep := range all {
		if strings.Contains(strings.ToLower(ep.Method+" "+ep.Path+" "+ep.Summary+" "+strings.Join(ep.Tags, " ")), kw) {
			results = append(results, ep)
		}
	}
	return results
}

func (o *OpenAPI) EndpointsByTag(tag string) []EndpointSummary {
	var results []EndpointSummary
	all := o.ListEndpoints()
	for _, ep := range all {
		for _, t := range ep.Tags {
			if t == tag {
				results = append(results, ep)
				break
			}
		}
	}
	return results
}

func (o *OpenAPI) EndpointsGroupedByTag() map[string][]EndpointSummary {
	groups := make(map[string][]EndpointSummary)
	all := o.ListEndpoints()
	for _, ep := range all {
		if len(ep.Tags) == 0 {
			groups["(no tag)"] = append(groups["(no tag)"], ep)
		} else {
			for _, t := range ep.Tags {
				groups[t] = append(groups[t], ep)
			}
		}
	}
	return groups
}

func (o *OpenAPI) GetEndpoint(path, method string) *EndpointDetail {
	paths, _ := o.raw["paths"].(map[string]any)
	if paths == nil {
		return nil
	}

	op, exists := paths[path]
	if !exists {
		fixed := fixMSYS2APIPath(path, mapKeys(paths))
		if fixed != path {
			return o.GetEndpoint(fixed, method)
		}
		return nil
	}
	methods, _ := op.(map[string]any)
	opData, exists := methods[strings.ToLower(method)]
	if !exists {
		return nil
	}
	opMap, _ := opData.(map[string]any)
	if opMap == nil {
		return nil
	}

	result := &EndpointDetail{
		Method:      strings.ToUpper(method),
		Path:        path,
		Summary:     str(opMap["summary"]),
		Description: str(opMap["description"]),
		OperationID: str(opMap["operationId"]),
		Tags:        toStringSlice(opMap["tags"]),
		Deprecated:  boolVal(opMap["deprecated"]),
	}

	// Parameters — resolve $ref at the top level only
	params, _ := opMap["parameters"].([]any)
	for _, raw := range params {
		pm, _ := raw.(map[string]any)
		if pm == nil {
			continue
		}
		if pm["in"] == "body" {
			continue
		}
		result.Parameters = append(result.Parameters, ParamDetail{
			Name:        str(pm["name"]),
			In:          str(pm["in"]),
			Required:    boolVal(pm["required"]),
			Type:        paramType(pm),
			Description: str(pm["description"]),
			Default:     pm["default"],
			Enum:        toAnySlice(pm["enum"]),
		})
	}

	// Request body — store raw schema for lazy resolution during formatting
	if o.isV2 {
		for _, raw := range params {
			pm, _ := raw.(map[string]any)
			if pm != nil && pm["in"] == "body" {
				result.RequestBody = &BodyDetail{
					Required:    boolVal(pm["required"]),
					Description: str(pm["description"]),
					MediaType:   "application/json",
					Schema:      pm["schema"],
				}
				break
			}
		}
	} else {
		rawRB := opMap["requestBody"]
		if rawRB != nil {
			rbm, _ := rawRB.(map[string]any)
			if rbm != nil {
				body := &BodyDetail{
					Required:    boolVal(rbm["required"]),
					Description: str(rbm["description"]),
				}
				content, _ := rbm["content"].(map[string]any)
				for mt, mediaRaw := range content {
					media, _ := mediaRaw.(map[string]any)
					body.MediaType = mt
					body.Schema = media["schema"]
					break
				}
				if body.MediaType == "" {
					body.MediaType = "application/json"
				}
				result.RequestBody = body
			}
		}
	}

	// Responses — store raw schemas for lazy formatting
	resps, _ := opMap["responses"].(map[string]any)
	for _, code := range sortedKeys(resps) {
		rawResp := resps[code]
		resp, _ := rawResp.(map[string]any)
		if resp == nil {
			continue
		}
		entry := ResponseDetail{
			Code:        code,
			Description: str(resp["description"]),
			MediaType:   "application/json",
		}
		if o.isV2 {
			entry.Schema = resp["schema"]
		} else {
			content, _ := resp["content"].(map[string]any)
			for mt, mediaRaw := range content {
				media, _ := mediaRaw.(map[string]any)
				entry.MediaType = mt
				entry.Schema = media["schema"]
				break
			}
		}
		result.Responses = append(result.Responses, entry)
	}

	return result
}

func (o *OpenAPI) GetSchema(name string) any {
	return o.resolveRef(o.schemaRefPath(name))
}

// UsesField finds all endpoints that reference a field in their schemas — O(n) raw traversal.
func (o *OpenAPI) UsesField(fieldName string) []EndpointSummary {
	var results []EndpointSummary
	paths, _ := o.raw["paths"].(map[string]any)
	for path, methods := range paths {
		ms, _ := methods.(map[string]any)
		for _, m := range httpMethods {
			op, ok := ms[m].(map[string]any)
			if !ok {
				continue
			}
			ep := EndpointSummary{
				Method:      strings.ToUpper(m),
				Path:        path,
				Summary:     str(op["summary"]),
				Tags:        toStringSlice(op["tags"]),
				OperationID: str(op["operationId"]),
			}
			if opReferencesField(op, fieldName, o) {
				results = append(results, ep)
			}
		}
	}
	return results
}

func opReferencesField(op map[string]any, name string, api *OpenAPI) bool {
	// Check parameter names and their schemas
	if params, ok := op["parameters"].([]any); ok {
		for _, p := range params {
			pm, _ := p.(map[string]any)
			if pm == nil {
				continue
			}
			if strings.EqualFold(str(pm["name"]), name) {
				return true
			}
			if schemaHasProp(pm["schema"], name, api) {
				return true
			}
		}
	}
	// Check request body
	if rb, ok := op["requestBody"].(map[string]any); ok {
		if bodyOrRespHasProp(rb, name, api) {
			return true
		}
	}
	// Check responses
	if resps, ok := op["responses"].(map[string]any); ok {
		for _, resp := range resps {
			if rm, ok := resp.(map[string]any); ok {
				if bodyOrRespHasProp(rm, name, api) {
					return true
				}
			}
		}
	}
	return false
}

func schemaHasProp(schema any, fieldName string, api *OpenAPI) bool {
	sm, _ := schema.(map[string]any)
	if sm == nil {
		return false
	}
	// Resolve $ref
	if ref, ok := sm["$ref"].(string); ok {
		resolved := api.resolveRef(ref)
		rm, _ := resolved.(map[string]any)
		if rm != nil {
			return schemaHasProp(rm, fieldName, api)
		}
	}
	props, _ := sm["properties"].(map[string]any)
	for pname := range props {
		if pname == fieldName {
			return true
		}
		if schemaHasProp(props[pname], fieldName, api) {
			return true
		}
	}
	return false
}

func bodyOrRespHasProp(body map[string]any, fieldName string, api *OpenAPI) bool {
	content, _ := body["content"].(map[string]any)
	for _, media := range content {
		mm, _ := media.(map[string]any)
		if mm != nil {
			if schemaHasProp(mm["schema"], fieldName, api) {
				return true
			}
		}
	}
	return false
}

// ── Formatting ─────────────────────────────────────────────────────

func formatInfo(info Info) string {
	var sb strings.Builder
	sb.WriteString("\n── API Info ──\n")
	fmt.Fprintf(&sb, "  Title:       %s\n", info.Title)
	fmt.Fprintf(&sb, "  Version:     %s\n", info.Version)
	if info.Description != "" {
		for _, line := range strings.Split(info.Description, "\n") {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
	}
	return sb.String()
}

func formatList(items []EndpointSummary, title string, compact bool) string {
	if len(items) == 0 {
		return fmt.Sprintf("No %s found.\n", strings.ToLower(title))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n── %s (%d) ──\n", title, len(items))

	if compact {
		for _, i := range items {
			fmt.Fprintf(&sb, "  %s %s", i.Method, i.Path)
			if i.Summary != "" {
				fmt.Fprintf(&sb, "  -- %s", i.Summary)
			}
			if i.Deprecated {
				fmt.Fprint(&sb, "  [!] DEPRECATED")
			}
			sb.WriteString("\n")
		}
	} else {
		maxM, maxP := 4, 4
		for _, i := range items {
			if len(i.Method) > maxM {
				maxM = len(i.Method)
			}
			if len(i.Path) > maxP {
				maxP = len(i.Path)
			}
		}
		for _, i := range items {
			m := fmt.Sprintf("%-*s", maxM, i.Method)
			p := fmt.Sprintf("%-*s", maxP, i.Path)
			s := i.Summary
			t := ""
			if len(i.Tags) > 0 {
				t = fmt.Sprintf(" [%s]", strings.Join(i.Tags, ", "))
			}
			d := ""
			if i.Deprecated {
				d = " [!] DEPRECATED"
			}
			fmt.Fprintf(&sb, "  %s  %s  %s%s%s\n", m, p, s, t, d)
		}
	}
	return sb.String()
}

func formatListGrouped(groups map[string][]EndpointSummary) string {
	if len(groups) == 0 {
		return "No endpoints found.\n"
	}
	var sb strings.Builder
	tags := sortedKeysStr(groups)
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	fmt.Fprintf(&sb, "\n── All Endpoints by Tag (%d) ──\n", total)
	for _, tag := range tags {
		items := groups[tag]
		fmt.Fprintf(&sb, "\n  [%s] (%d):\n", tag, len(items))
		for _, item := range items {
			fmt.Fprintf(&sb, "    %s %s", item.Method, item.Path)
			if item.Summary != "" {
				fmt.Fprintf(&sb, "  -- %s", item.Summary)
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (o *OpenAPI) formatEndpoint(ep *EndpointDetail, depth int, paramsOnly, requestOnly, responseOnly, fieldsOnly bool) string {
	if ep == nil {
		return "Endpoint not found.\n"
	}

	showAll := !paramsOnly && !requestOnly && !responseOnly
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("============================================================\n")
	header := fmt.Sprintf("%s %s", ep.Method, ep.Path)
	if ep.Summary != "" {
		header += "  -- " + ep.Summary
	}
	sb.WriteString(header + "\n")
	if ep.Deprecated {
		sb.WriteString("  [!] DEPRECATED\n")
	}
	sb.WriteString("============================================================\n")

	metaParts := []string{}
	if len(ep.Tags) > 0 {
		metaParts = append(metaParts, "Tags: "+strings.Join(ep.Tags, ", "))
	}
	if ep.OperationID != "" {
		metaParts = append(metaParts, "ID: "+ep.OperationID)
	}
	if len(metaParts) > 0 {
		fmt.Fprintf(&sb, "  %s\n", strings.Join(metaParts, "  |  "))
	}
	if ep.Description != "" {
		fmt.Fprintf(&sb, "  %s\n", ep.Description)
	}
	sb.WriteString("\n")

	if showAll || paramsOnly {
		if len(ep.Parameters) > 0 {
			sb.WriteString("── Parameters ──\n")
			for _, p := range ep.Parameters {
				reqMark := " "
				if p.Required {
					reqMark = "*"
				}
				loc := map[string]string{"query": "?", "path": ":", "header": "H", "cookie": "C"}[p.In]
				if loc == "" {
					loc = p.In
				}
				enumS := ""
				if len(p.Enum) > 0 {
					parts := make([]string, len(p.Enum))
					for i, e := range p.Enum {
						parts[i] = fmt.Sprint(e)
					}
					enumS = " [enum: " + strings.Join(parts, ", ") + "]"
				}
				defS := ""
				if p.Default != nil {
					defS = fmt.Sprintf("  default: %v", p.Default)
				}
				descS := ""
				if p.Description != "" {
					descS = "  -- " + p.Description
				}
				fmt.Fprintf(&sb, "  %s%s: %s  (%s)%s%s%s\n", reqMark, loc, p.Name, p.Type, enumS, defS, descS)
			}
			sb.WriteString("\n")
		}
	}

	if showAll || requestOnly {
		if ep.RequestBody != nil && ep.RequestBody.Schema != nil {
			reqNote := " (required)"
			if !ep.RequestBody.Required {
				reqNote = " (optional)"
			}
			fmt.Fprintf(&sb, "── Request Body (%s)%s ──\n", ep.RequestBody.MediaType, reqNote)
			if ep.RequestBody.Description != "" {
				fmt.Fprintf(&sb, "  %s\n", ep.RequestBody.Description)
			}
			sb.WriteString(o.formatSchemaFields(ep.RequestBody.Schema, 0, depth, fieldsOnly))
			sb.WriteString("\n")
		}
	}

	if showAll || responseOnly {
		for _, resp := range ep.Responses {
			descStr := ""
			if resp.Description != "" {
				descStr = " -- " + resp.Description
			}
			fmt.Fprintf(&sb, "── Response %s (%s)%s ──\n", resp.Code, resp.MediaType, descStr)
			if resp.Schema != nil {
				sb.WriteString(o.formatSchemaFields(resp.Schema, 0, depth, fieldsOnly))
			} else {
				sb.WriteString("  (no content)\n")
			}
			sb.WriteString("\n")
		}
	}

	if showAll && !fieldsOnly {
		sb.WriteString("── Legend ──\n")
		sb.WriteString("  * = required  ? = query  : = path  H = header  C = cookie\n")
	}
	return sb.String()
}

func formatSchema(name string, schema any, depth int, fieldsOnly bool) string {
	if schema == nil {
		return fmt.Sprintf("\nSchema '%s' not found in spec.\n", name)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n── Schema: %s ──\n", name)
	sb.WriteString(formatSchemaFields(schema, 0, depth, fieldsOnly))
	return sb.String()
}

// formatSchemaFields is a standalone version preserved for external callers (e.g. schema command).
// It does NOT resolve $ref — callers must pass resolved schemas.
func formatSchemaFields(schema any, indent int, maxDepth int, fieldsOnly bool) string {
	var sb strings.Builder
	pf := strings.Repeat("  ", indent)

	sm, ok := schema.(map[string]any)
	if !ok {
		fmt.Fprintf(&sb, "%s%v\n", pf, schema)
		return sb.String()
	}

	// Handle composed schemas
	for _, key := range []string{"allOf", "oneOf", "anyOf"} {
		if subs, ok := sm[key].([]any); ok {
			fmt.Fprintf(&sb, "%s[%s]\n", pf, key)
			if maxDepth > 0 && indent >= maxDepth {
				fmt.Fprintf(&sb, "%s  ... (max depth reached)\n", pf)
			} else {
				for _, sub := range subs {
					sb.WriteString(formatSchemaFields(sub, indent+1, maxDepth, fieldsOnly))
				}
			}
			return sb.String()
		}
	}

	stype := str(sm["type"])
	if stype == "" {
		stype = "object"
	}

	if stype == "object" {
		props, _ := sm["properties"].(map[string]any)
		required := toStringSet(sm["required"])
		additional := sm["additionalProperties"]

		if len(props) == 0 {
			if additional != nil {
				if am, ok := additional.(map[string]any); ok {
					fmt.Fprintf(&sb, "%s{string: %s}  (map)\n", pf, str(am["type"]))
				} else if b, ok := additional.(bool); ok && b {
					fmt.Fprint(&sb, pf+"{string: any}  (map)\n")
				}
			} else {
				fmt.Fprintf(&sb, "%s{}  (empty object)\n", pf)
			}
			return sb.String()
		}

		if maxDepth > 0 && indent >= maxDepth {
			fmt.Fprintf(&sb, "%s... (max depth reached, %d fields)\n", pf, len(props))
		} else {
			for _, pname := range sortedKeys(props) {
				sb.WriteString(formatProp(pname, props[pname], indent, required[pname], maxDepth, fieldsOnly))
			}
		}

		if b, ok := additional.(bool); ok && b {
			fmt.Fprintf(&sb, "%s...  (additional properties allowed)\n", pf)
		} else if am, ok := additional.(map[string]any); ok {
			fmt.Fprintf(&sb, "%s...  (additional: %s)\n", pf, str(am["type"]))
		}
	} else if stype == "array" {
		items := sm["items"]
		if items != nil {
			if im, ok := items.(map[string]any); ok {
				itype := str(im["type"])
				if itype == "object" && im["properties"] != nil {
					fmt.Fprintf(&sb, "%s[]\n", pf)
					if maxDepth > 0 && indent >= maxDepth {
						fmt.Fprintf(&sb, "%s  ... (max depth reached)\n", pf)
					} else {
						sb.WriteString(formatSchemaFields(items, indent+1, maxDepth, fieldsOnly))
					}
				} else if itype != "" {
					fmt.Fprintf(&sb, "%s[]  (of %s)\n", pf, itype)
				} else if ref, ok := im["$ref"].(string); ok {
					refName := ref[strings.LastIndex(ref, "/")+1:]
					fmt.Fprintf(&sb, "%s[]  (of %s)\n", pf, refName)
				} else {
					fmt.Fprintf(&sb, "%s[]\n", pf)
				}
			}
		} else {
			fmt.Fprintf(&sb, "%s[]\n", pf)
		}
	} else {
		extras := []string{}
		if enum, ok := sm["enum"].([]any); ok {
			vals := make([]string, len(enum))
			for i, e := range enum {
				vals[i] = fmt.Sprint(e)
			}
			extras = append(extras, "enum: "+strings.Join(vals, ", "))
		}
		if f := str(sm["format"]); f != "" {
			extras = append(extras, f)
		}
		if d := sm["default"]; d != nil {
			extras = append(extras, fmt.Sprintf("default: %v", d))
		}
		if boolVal(sm["nullable"]) {
			extras = append(extras, "nullable")
		}
		line := pf + stype
		if len(extras) > 0 {
			line += "  (" + strings.Join(extras, " | ") + ")"
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// formatProp is the standalone version (no $ref resolution).
func formatProp(name string, prop any, indent int, required bool, maxDepth int, fieldsOnly bool) string {
	var sb strings.Builder
	pf := strings.Repeat("  ", indent)
	mark := " "
	if required {
		mark = "*"
	}

	pm, ok := prop.(map[string]any)
	if !ok {
		fmt.Fprintf(&sb, "%s%s%s: %v\n", pf, name, mark, prop)
		return sb.String()
	}

	ptype := str(pm["type"])
	if ptype == "" {
		ptype = "object"
	}
	desc := str(pm["description"])
	def := pm["default"]
	enum := toAnySlice(pm["enum"])
	fmtStr := str(pm["format"])
	nullable := boolVal(pm["nullable"])

	var extras []string
	if fmtStr != "" {
		extras = append(extras, fmtStr)
	}
	if len(enum) > 0 {
		vals := make([]string, len(enum))
		for i, e := range enum {
			vals[i] = fmt.Sprint(e)
		}
		extras = append(extras, "enum: "+strings.Join(vals, ", "))
	}
	if def != nil {
		extras = append(extras, fmt.Sprintf("default: %v", def))
	}
	if nullable {
		extras = append(extras, "nullable")
	}

	meta := ""
	if len(extras) > 0 {
		meta = " (" + strings.Join(extras, " | ") + ")"
	}
	hint := ""
	if desc != "" {
		hint = "  -- " + desc
	}

	if fieldsOnly {
		fmt.Fprintf(&sb, "%s%s%s: %s%s\n", pf, name, mark, ptype, hint)
		return sb.String()
	}

	if ptype == "object" {
		hasContent := pm["properties"] != nil || pm["allOf"] != nil || pm["oneOf"] != nil || pm["anyOf"] != nil
		fmt.Fprintf(&sb, "%s%s%s: object%s%s\n", pf, name, mark, meta, hint)
		if hasContent {
			if maxDepth > 0 && indent+1 >= maxDepth {
				fmt.Fprintf(&sb, "%s  ... (max depth reached)\n", pf)
			} else {
				sb.WriteString(formatSchemaFields(prop, indent+1, maxDepth, fieldsOnly))
			}
		}
	} else if ptype == "array" {
		fmt.Fprintf(&sb, "%s%s%s: array%s%s\n", pf, name, mark, meta, hint)
		items := pm["items"]
		if im, ok := items.(map[string]any); ok {
			itype := str(im["type"])
			if itype == "object" && im["properties"] != nil {
				if maxDepth > 0 && indent+1 >= maxDepth {
					fmt.Fprintf(&sb, "%s  ... (max depth reached)\n", pf)
				} else {
					sb.WriteString(formatSchemaFields(items, indent+1, maxDepth, fieldsOnly))
				}
			} else if itype != "" {
				ifmt := str(im["format"])
				idesc := str(im["description"])
				ihint := ""
				if idesc != "" {
					ihint = "  -- " + idesc
				}
				innerMeta := ""
				if ifmt != "" {
					innerMeta = " (" + ifmt + ")"
				}
				fmt.Fprintf(&sb, "%s  [%s%s]%s\n", pf, itype, innerMeta, ihint)
			} else if ref, ok := im["$ref"].(string); ok {
				refName := ref[strings.LastIndex(ref, "/")+1:]
				fmt.Fprintf(&sb, "%s  [%s]\n", pf, refName)
			}
		}
	} else {
		fmt.Fprintf(&sb, "%s%s%s: %s%s%s\n", pf, name, mark, ptype, meta, hint)
	}

	return sb.String()
}

// ── Lazy $ref resolution formatting (method-based) ────────────────

func (o *OpenAPI) formatSchemaFields(schema any, indent int, maxDepth int, fieldsOnly bool) string {
	sm, ok := schema.(map[string]any)
	if !ok {
		return fmt.Sprintf("%s%v\n", strings.Repeat("  ", indent), schema)
	}

	// Lazy $ref resolution
	if ref, ok := sm["$ref"].(string); ok {
		resolved := o.resolveRef(ref)
		if resolved != nil {
			return o.formatSchemaFields(resolved, indent, maxDepth, fieldsOnly)
		}
	}

	// Handle composed schemas
	for _, key := range []string{"allOf", "oneOf", "anyOf"} {
		if subs, ok := sm[key].([]any); ok {
			var sb strings.Builder
			pf := strings.Repeat("  ", indent)
			fmt.Fprintf(&sb, "%s[%s]\n", pf, key)
			if maxDepth > 0 && indent >= maxDepth {
				fmt.Fprintf(&sb, "%s  ... (max depth reached)\n", pf)
			} else {
				for _, sub := range subs {
					sb.WriteString(o.formatSchemaFields(sub, indent+1, maxDepth, fieldsOnly))
				}
			}
			return sb.String()
		}
	}

	var sb strings.Builder
	pf := strings.Repeat("  ", indent)

	stype := str(sm["type"])
	if stype == "" {
		stype = "object"
	}

	if stype == "object" {
		props, _ := sm["properties"].(map[string]any)
		required := toStringSet(sm["required"])
		additional := sm["additionalProperties"]

		if len(props) == 0 {
			if additional != nil {
				if am, ok := additional.(map[string]any); ok {
					fmt.Fprintf(&sb, "%s{string: %s}  (map)\n", pf, str(am["type"]))
				} else if b, ok := additional.(bool); ok && b {
					fmt.Fprint(&sb, pf+"{string: any}  (map)\n")
				}
			} else {
				fmt.Fprintf(&sb, "%s{}  (empty object)\n", pf)
			}
			return sb.String()
		}

		if maxDepth > 0 && indent >= maxDepth {
			fmt.Fprintf(&sb, "%s... (max depth reached, %d fields)\n", pf, len(props))
		} else {
			for _, pname := range sortedKeys(props) {
				sb.WriteString(o.formatProp(pname, props[pname], indent, required[pname], maxDepth, fieldsOnly))
			}
		}

		if b, ok := additional.(bool); ok && b {
			fmt.Fprintf(&sb, "%s...  (additional properties allowed)\n", pf)
		} else if am, ok := additional.(map[string]any); ok {
			fmt.Fprintf(&sb, "%s...  (additional: %s)\n", pf, str(am["type"]))
		}
	} else if stype == "array" {
		items := sm["items"]
		if items != nil {
			if im, ok := items.(map[string]any); ok {
				itype := str(im["type"])
				if itype == "object" && im["properties"] != nil {
					fmt.Fprintf(&sb, "%s[]\n", pf)
					if maxDepth > 0 && indent >= maxDepth {
						fmt.Fprintf(&sb, "%s  ... (max depth reached)\n", pf)
					} else {
						sb.WriteString(o.formatSchemaFields(items, indent+1, maxDepth, fieldsOnly))
					}
				} else if itype != "" {
					fmt.Fprintf(&sb, "%s[]  (of %s)\n", pf, itype)
				} else if ref, ok := im["$ref"].(string); ok {
					resolved := o.resolveRef(ref)
					if resolved != nil {
						fmt.Fprintf(&sb, "%s[]\n", pf)
						sb.WriteString(o.formatSchemaFields(resolved, indent+1, maxDepth, fieldsOnly))
					} else {
						refName := ref[strings.LastIndex(ref, "/")+1:]
						fmt.Fprintf(&sb, "%s[]  (of %s)\n", pf, refName)
					}
				} else {
					fmt.Fprintf(&sb, "%s[]\n", pf)
				}
			}
		} else {
			fmt.Fprintf(&sb, "%s[]\n", pf)
		}
	} else {
		extras := []string{}
		if enum, ok := sm["enum"].([]any); ok {
			vals := make([]string, len(enum))
			for i, e := range enum {
				vals[i] = fmt.Sprint(e)
			}
			extras = append(extras, "enum: "+strings.Join(vals, ", "))
		}
		if f := str(sm["format"]); f != "" {
			extras = append(extras, f)
		}
		if d := sm["default"]; d != nil {
			extras = append(extras, fmt.Sprintf("default: %v", d))
		}
		if boolVal(sm["nullable"]) {
			extras = append(extras, "nullable")
		}
		line := pf + stype
		if len(extras) > 0 {
			line += "  (" + strings.Join(extras, " | ") + ")"
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func (o *OpenAPI) formatProp(name string, prop any, indent int, required bool, maxDepth int, fieldsOnly bool) string {
	pf := strings.Repeat("  ", indent)
	mark := " "
	if required {
		mark = "*"
	}

	pm, ok := prop.(map[string]any)
	if !ok {
		return fmt.Sprintf("%s%s%s: %v\n", pf, name, mark, prop)
	}

	// Lazy $ref resolution
	if ref, ok := pm["$ref"].(string); ok {
		resolved := o.resolveRef(ref)
		if resolved != nil {
			return o.formatProp(name, resolved, indent, required, maxDepth, fieldsOnly)
		}
	}

	var sb strings.Builder

	ptype := str(pm["type"])
	if ptype == "" {
		ptype = "object"
	}
	desc := str(pm["description"])
	def := pm["default"]
	enum := toAnySlice(pm["enum"])
	fmtStr := str(pm["format"])
	nullable := boolVal(pm["nullable"])

	var extras []string
	if fmtStr != "" {
		extras = append(extras, fmtStr)
	}
	if len(enum) > 0 {
		vals := make([]string, len(enum))
		for i, e := range enum {
			vals[i] = fmt.Sprint(e)
		}
		extras = append(extras, "enum: "+strings.Join(vals, ", "))
	}
	if def != nil {
		extras = append(extras, fmt.Sprintf("default: %v", def))
	}
	if nullable {
		extras = append(extras, "nullable")
	}

	meta := ""
	if len(extras) > 0 {
		meta = " (" + strings.Join(extras, " | ") + ")"
	}
	hint := ""
	if desc != "" {
		hint = "  -- " + desc
	}

	if fieldsOnly {
		return fmt.Sprintf("%s%s%s: %s%s\n", pf, name, mark, ptype, hint)
	}

	if ptype == "object" {
		hasContent := pm["properties"] != nil || pm["allOf"] != nil || pm["oneOf"] != nil || pm["anyOf"] != nil
		fmt.Fprintf(&sb, "%s%s%s: object%s%s\n", pf, name, mark, meta, hint)
		if hasContent {
			if maxDepth > 0 && indent+1 >= maxDepth {
				fmt.Fprintf(&sb, "%s  ... (max depth reached)\n", pf)
			} else {
				sb.WriteString(o.formatSchemaFields(prop, indent+1, maxDepth, fieldsOnly))
			}
		}
	} else if ptype == "array" {
		fmt.Fprintf(&sb, "%s%s%s: array%s%s\n", pf, name, mark, meta, hint)
		items := pm["items"]
		if im, ok := items.(map[string]any); ok {
			itype := str(im["type"])
			if itype == "object" && im["properties"] != nil {
				if maxDepth > 0 && indent+1 >= maxDepth {
					fmt.Fprintf(&sb, "%s  ... (max depth reached)\n", pf)
				} else {
					sb.WriteString(o.formatSchemaFields(items, indent+1, maxDepth, fieldsOnly))
				}
			} else if itype != "" {
				ifmt := str(im["format"])
				idesc := str(im["description"])
				ihint := ""
				if idesc != "" {
					ihint = "  -- " + idesc
				}
				innerMeta := ""
				if ifmt != "" {
					innerMeta = " (" + ifmt + ")"
				}
				fmt.Fprintf(&sb, "%s  [%s%s]%s\n", pf, itype, innerMeta, ihint)
			} else if ref, ok := im["$ref"].(string); ok {
				resolved := o.resolveRef(ref)
				if resolved != nil {
					if rm, ok := resolved.(map[string]any); ok && rm["properties"] != nil {
						fmt.Fprintf(&sb, "%s[]\n", pf)
						if maxDepth > 0 && indent+1 >= maxDepth {
							fmt.Fprintf(&sb, "%s  ... (max depth reached)\n", pf)
						} else {
							sb.WriteString(o.formatSchemaFields(resolved, indent+1, maxDepth, fieldsOnly))
						}
					} else {
						refName := ref[strings.LastIndex(ref, "/")+1:]
						fmt.Fprintf(&sb, "%s  [%s]\n", pf, refName)
					}
				} else {
					refName := ref[strings.LastIndex(ref, "/")+1:]
					fmt.Fprintf(&sb, "%s  [%s]\n", pf, refName)
				}
			}
		}
	} else {
		fmt.Fprintf(&sb, "%s%s%s: %s%s%s\n", pf, name, mark, ptype, meta, hint)
	}

	return sb.String()
}

// ── JSON Output ────────────────────────────────────────────────────

func jsonOutput(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("Error marshaling JSON: %v\n", err)
	}
	return string(data) + "\n"
}

// ── Helpers ────────────────────────────────────────────────────────

func str(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func boolVal(v any) bool {
	b, _ := v.(bool)
	return b
}

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	arr, _ := v.([]any)
	if arr == nil {
		return nil
	}
	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = fmt.Sprint(item)
	}
	return result
}

func toAnySlice(v any) []any {
	if v == nil {
		return nil
	}
	arr, _ := v.([]any)
	return arr
}

func toStringSet(v any) map[string]bool {
	result := make(map[string]bool)
	sl := toAnySlice(v)
	for _, item := range sl {
		result[fmt.Sprint(item)] = true
	}
	return result
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysStr(m map[string][]EndpointSummary) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func paramType(p map[string]any) string {
	if schema, ok := p["schema"].(map[string]any); ok {
		return str(schema["type"])
	}
	if t := str(p["type"]); t != "" {
		return t
	}
	return "string"
}
