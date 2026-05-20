package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const walkSchemaMaxDepth = 64

// ── Write Operations ───────────────────────────────────────────────

func UpsertEndpoint(o *OpenAPI, path, method string, operation map[string]any) (result string, warnings []string) {
	paths, _ := o.raw["paths"].(map[string]any)
	if paths == nil {
		paths = make(map[string]any)
		o.raw["paths"] = paths
	}

	fixed := fixMSYS2APIPath(path, mapKeys(paths))
	if fixed != path {
		path = fixed
	}

	methodLower := strings.ToLower(method)

	methods, _ := paths[path].(map[string]any)
	if methods == nil {
		methods = make(map[string]any)
		paths[path] = methods
	}

	isUpdate := methods[methodLower] != nil
	methods[methodLower] = operation

	warnings = validateEndpointQuality(path, methodLower, operation)
	if isUpdate {
		result = fmt.Sprintf("  %s %s  updated", strings.ToUpper(method), path)
	} else {
		result = fmt.Sprintf("  %s %s  new", strings.ToUpper(method), path)
	}
	return
}

func RemoveEndpoint(o *OpenAPI, path, method string) string {
	paths, _ := o.raw["paths"].(map[string]any)
	if paths == nil {
		return fmt.Sprintf("  Path %s not found.", path)
	}

	fixed := fixMSYS2APIPath(path, mapKeys(paths))
	if fixed != path {
		path = fixed
	}

	if _, exists := paths[path]; !exists {
		return fmt.Sprintf("  Path %s not found.", path)
	}

	if method != "" {
		methodLower := strings.ToLower(method)
		methods, _ := paths[path].(map[string]any)
		if methods != nil {
			if _, exists := methods[methodLower]; exists {
				delete(methods, methodLower)
				if len(methods) == 0 {
					delete(paths, path)
				}
				return fmt.Sprintf("  %s %s  removed", strings.ToUpper(method), path)
			}
		}
		return fmt.Sprintf("  %s %s  not found.", strings.ToUpper(method), path)
	}

	delete(paths, path)
	return fmt.Sprintf("  All methods on %s removed.", path)
}

func UpsertSchema(o *OpenAPI, name string, schemaData any) (result string, warnings []string) {
	if o.isV2 {
		if o.raw["definitions"] == nil {
			o.raw["definitions"] = make(map[string]any)
		}
		defs, _ := o.raw["definitions"].(map[string]any)
		isUpdate := defs[name] != nil
		defs[name] = defensiveCopy(schemaData)
		if isUpdate {
			result = fmt.Sprintf("  Schema %s  updated", name)
		} else {
			result = fmt.Sprintf("  Schema %s  added", name)
		}
	} else {
		if o.raw["components"] == nil {
			o.raw["components"] = make(map[string]any)
		}
		components, _ := o.raw["components"].(map[string]any)
		if components["schemas"] == nil {
			components["schemas"] = make(map[string]any)
		}
		schemas, _ := components["schemas"].(map[string]any)
		isUpdate := schemas[name] != nil
		schemas[name] = defensiveCopy(schemaData)
		if isUpdate {
			result = fmt.Sprintf("  Schema %s  updated", name)
		} else {
			result = fmt.Sprintf("  Schema %s  added", name)
		}
	}

	warnings = validateSchemaQuality(name, schemaData)
	return
}

func defensiveCopy(val any) any {
	switch v := val.(type) {
	case map[string]any:
		r := make(map[string]any, len(v))
		for k, x := range v {
			r[k] = defensiveCopy(x)
		}
		return r
	case []any:
		r := make([]any, len(v))
		for i, x := range v {
			r[i] = defensiveCopy(x)
		}
		return r
	default:
		return val
	}
}

func SaveSpec(o *OpenAPI, filepathArg string) error {
	// Clean up stale .tmp from previous crash
	tryCleanStaleTmp(filepathArg)

	// Backup original if it exists
	tryBackup(filepathArg)

	// Atomic write: write to .tmp then rename
	tmp := filepathArg + ".tmp"
	data, err := json.MarshalIndent(o.raw, "", "  ")
	if err != nil {
		return err
	}

	// Preserve original file permissions on Unix
	mode := os.FileMode(0644)
	if fi, err := os.Stat(filepathArg); err == nil {
		mode = fi.Mode()
	}
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}

	// Cross-device safe rename: try Rename first, fallback to copy+delete
	if err := os.Rename(tmp, filepathArg); err != nil {
		if err2 := crossDeviceCopyFallback(tmp, filepathArg, mode); err2 != nil {
			os.Remove(tmp)
			return err2
		}
	}
	return nil
}

func tryCleanStaleTmp(path string) {
	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); err == nil {
		os.Remove(tmp)
	}
}

func tryBackup(path string) {
	if _, err := os.Stat(path); err != nil {
		return
	}
	bak := path + ".bak"
	_ = copyFile(path, bak)
}

func crossDeviceCopyFallback(src, dst string, mode os.FileMode) error {
	if err := copyFile(src, dst); err != nil {
		return err
	}
	os.Chmod(dst, mode)
	os.Remove(src)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ── Quality Validation ─────────────────────────────────────────────

var cjkRe = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{f900}-\x{faff}]`)

func hasCJK(text string) bool {
	return cjkRe.MatchString(text)
}

var enumValueRe = regexp.MustCompile(`\d+\s*=`) // matches "0=", "1=", etc

func hasEnumDocs(desc string) bool {
	return enumValueRe.MatchString(desc)
}

func validateDescription(label string, value string, warnings *[]string) {
	if value == "" || value == "\"\"" {
		*warnings = append(*warnings, fmt.Sprintf("    [!] %s: description is empty", label))
	} else if !hasCJK(value) {
		*warnings = append(*warnings, fmt.Sprintf("    [~] %s: description should include Chinese notes (current: %.40s)", label, value))
	}
}

func validateFieldEnumDocs(label string, pvalue any, warnings *[]string) {
	pm, ok := pvalue.(map[string]any)
	if !ok {
		return
	}

	target := pm
	if schema, ok := pm["schema"].(map[string]any); ok {
		if _, hasEnum := schema["enum"]; hasEnum {
			target = schema
		} else if _, hasType := schema["type"]; hasType {
			target = schema
		}
	}

	hasEnum := target["enum"] != nil
	stype := str(target["type"])
	isIntegerLike := stype == "integer" || stype == "number"

	desc := str(target["description"])
	if desc == "" {
		desc = str(pm["description"])
	}

	if hasEnum && !hasEnumDocs(desc) {
		*warnings = append(*warnings, fmt.Sprintf(
			"    [!] %s: enum %v missing value meaning docs, description should include \"value=meaning\" mappings",
			label, target["enum"],
		))
	} else if isIntegerLike && !hasEnum && desc != "" && !hasEnumDocs(desc) {
		*warnings = append(*warnings, fmt.Sprintf(
			"    [~] %s: integer field should document enum values in description",
			label,
		))
	}
}

func validateRequiredDocs(label string, required []any, props map[string]any, warnings *[]string) {
	if len(props) == 0 {
		return
	}
	if len(required) == 0 {
		*warnings = append(*warnings, fmt.Sprintf("    [!] %s: object schema has properties but no required array", label))
		return
	}
	for _, rfield := range required {
		rfStr := fmt.Sprint(rfield)
		if _, exists := props[rfStr]; !exists {
			*warnings = append(*warnings, fmt.Sprintf("    [!] %s: required field '%s' not found in properties", label, rfStr))
		} else if prop, ok := props[rfStr].(map[string]any); ok {
			desc := str(prop["description"])
			if !strings.Contains(desc, "\u5fc5\u586b") {
				*warnings = append(*warnings, fmt.Sprintf("    [~] %s: required field '%s' description should mark \uff08\u5fc5\u586b\uff09", label, rfStr))
			}
		}
	}
}

func walkSchema(ctx string, schema any, warnings *[]string, seen map[string]bool, depth int) {
	if depth > walkSchemaMaxDepth {
		return
	}
	sm, ok := schema.(map[string]any)
	if !ok {
		return
	}
	if seen == nil {
		seen = make(map[string]bool)
	}

	// Follow $ref (skip already-seen refs)
	if ref, ok := sm["$ref"].(string); ok {
		if seen[ref] {
			return
		}
		seen[ref] = true
		// The actual validation is done on the raw definition — skip
		return
	}

	// Traverse allOf / oneOf / anyOf
	for _, composed := range [][]string{{"allOf"}, {"oneOf"}, {"anyOf"}} {
		if subs, ok := sm[composed[0]].([]any); ok {
			for i, sub := range subs {
				walkSchema(fmt.Sprintf("%s %s[%d]", ctx, composed[0], i), sub, warnings, seen, depth+1)
			}
			return
		}
	}

	stype := str(sm["type"])
	props, _ := sm["properties"].(map[string]any)
	req := toAnySlice(sm["required"])

	if stype == "object" && len(props) > 0 {
		validateRequiredDocs(ctx, req, props, warnings)
	}

	for pname, pvalue := range props {
		if pm, ok := pvalue.(map[string]any); ok {
			pctx := fmt.Sprintf("%s field %s", ctx, pname)
			validateDescription(pctx, str(pm["description"]), warnings)
			validateFieldEnumDocs(pctx, pvalue, warnings)

			isRequired := false
			for _, r := range req {
				if fmt.Sprint(r) == pname {
					isRequired = true
					break
				}
			}
			if !isRequired {
				pdesc := str(pm["description"])
				if pdesc != "" && !strings.Contains(pdesc, "\u9009\u586b") && !strings.Contains(pdesc, "\u5fc5\u586b") {
					*warnings = append(*warnings, fmt.Sprintf("    [~] %s: optional field should mark \uff08\u9009\u586b\uff09", pctx))
				}
			}

			if str(pm["type"]) == "object" || pm["properties"] != nil {
				walkSchema(pctx, pvalue, warnings, seen, depth+1)
			}
			if items, ok := pm["items"].(map[string]any); ok {
				if items["properties"] != nil || items["$ref"] != nil {
					walkSchema(pctx+"[]", items, warnings, seen, depth+1)
				}
			}
			// Recurse into composed sub-schemas in properties
			for _, key := range []string{"allOf", "oneOf", "anyOf"} {
				if subs, ok := pm[key].([]any); ok {
					for i, sub := range subs {
						walkSchema(fmt.Sprintf("%s field %s %s[%d]", ctx, pname, key, i), sub, warnings, seen, depth+1)
					}
				}
			}
		}
	}
}

func validateEndpointQuality(path, method string, op map[string]any) []string {
	var warnings []string
	ctx := fmt.Sprintf("%s %s", strings.ToUpper(method), path)

	validateDescription(fmt.Sprintf("%s summary", ctx), str(op["summary"]), &warnings)
	validateDescription(fmt.Sprintf("%s description", ctx), str(op["description"]), &warnings)

	if params, ok := op["parameters"].([]any); ok {
		for _, p := range params {
			pm, _ := p.(map[string]any)
			if pm == nil {
				continue
			}
			pname := str(pm["name"])
			pctx := fmt.Sprintf("%s parameter %s", ctx, pname)
			validateDescription(pctx, str(pm["description"]), &warnings)
			validateFieldEnumDocs(pctx, p, &warnings)
		}
	}

	if responses, ok := op["responses"].(map[string]any); ok {
		for code, resp := range responses {
			rctx := fmt.Sprintf("%s response %s", ctx, code)
			if rm, ok := resp.(map[string]any); ok {
				validateDescription(rctx, str(rm["description"]), &warnings)
				validateSchemaFields(rctx, resp, &warnings)
			}
		}
	}

	if rb, ok := op["requestBody"].(map[string]any); ok {
		validateDescription(fmt.Sprintf("%s requestBody", ctx), str(rb["description"]), &warnings)
		validateSchemaFields(fmt.Sprintf("%s requestBody", ctx), rb, &warnings)
	}

	return warnings
}

func validateSchemaFields(ctx string, parent any, warnings *[]string) {
	pm, ok := parent.(map[string]any)
	if !ok {
		return
	}
	content, _ := pm["content"].(map[string]any)
	for _, ct := range content {
		if cm, ok := ct.(map[string]any); ok {
			if schema := cm["schema"]; schema != nil {
				walkSchema(ctx, schema, warnings, nil, 0)
			}
		}
	}
}

func validateSchemaQuality(name string, schema any) []string {
	var warnings []string
	walkSchema("Schema "+name, schema, &warnings, nil, 0)
	return warnings
}
