// Package newfeature — factory_template.go embeds and renders factory stub templates.
//
// Two templates are embedded:
//   - factory.ts.tmpl — TypeScript factory stub (REQ-NS-01)
//   - factory.js.tmpl — JavaScript factory stub (REQ-NS-06)
//
// Templates are rendered with text/template using a data struct that exposes
// the schematic Name plus helper functions (pascalCase, camelCase).
// The helpers are safe — they do not call into tsident (which handles TS
// identifier escaping at codegen time). Template rendering only embeds the
// name into a comment and identifier; the generator logic in tsgen.go handles
// the real identifier escaping for property names.
//
// ADR-025: NO text/template for .d.ts codegen (silent-injection risk). For
// factory stubs the risk is different — the name appears in a comment and a
// function declaration, and the caller has already validated the name via
// validate.RejectMetachars. The template is simple string substitution.
package newfeature

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

//go:embed stub_templates/factory.ts.tmpl
var factoryTSTmpl string

//go:embed stub_templates/factory.js.tmpl
var factoryJSTmpl string

// factoryTemplateFuncs provides simple string helpers for factory templates.
// These are NOT for TS identifier escaping (that is tsident's job) — they are
// just for rendering readable stub function names in the template comments and
// export declarations.
var factoryTemplateFuncs = template.FuncMap{
	// pascalCase converts "my-schematic" → "MySchematic"
	"pascalCase": func(s string) string {
		parts := strings.FieldsFunc(s, func(r rune) bool {
			return r == '-' || r == '_' || r == ' '
		})
		var sb strings.Builder
		for _, p := range parts {
			if len(p) > 0 {
				sb.WriteString(strings.ToUpper(p[:1]) + p[1:])
			}
		}
		return sb.String()
	},
	// camelCase converts "my-schematic" → "mySchematic"
	"camelCase": func(s string) string {
		parts := strings.FieldsFunc(s, func(r rune) bool {
			return r == '-' || r == '_' || r == ' '
		})
		var sb strings.Builder
		for i, p := range parts {
			if len(p) == 0 {
				continue
			}
			if i == 0 {
				sb.WriteString(strings.ToLower(p))
			} else {
				sb.WriteString(strings.ToUpper(p[:1]) + p[1:])
			}
		}
		return sb.String()
	},
}

// LoadFactoryTemplate returns the raw template string for the given language.
//
// Supported languages: "ts" (TypeScript), "js" (JavaScript).
// Returns ErrCodeInvalidLanguage for any other value.
func LoadFactoryTemplate(language string) (string, error) {
	switch language {
	case "ts":
		return factoryTSTmpl, nil
	case "js":
		return factoryJSTmpl, nil
	default:
		return "", &errs.Error{
			Code:    errs.ErrCodeInvalidLanguage,
			Op:      "factory_template.load",
			Message: "--language '" + language + "': unsupported; valid values: ts, js",
		}
	}
}

// factoryData is the data struct passed to factory templates.
type factoryData struct {
	Name string
}

// RenderFactoryTemplate renders the factory stub template for the given language
// with the given schematic name substituted.
//
// Returns ErrCodeInvalidLanguage for unsupported languages.
func RenderFactoryTemplate(language, name string) ([]byte, error) {
	raw, err := LoadFactoryTemplate(language)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("factory").Funcs(factoryTemplateFuncs).Parse(raw)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "factory_template.render",
			Message: "failed to parse factory template",
			Cause:   err,
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, factoryData{Name: name}); err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "factory_template.render",
			Message: "failed to render factory template for schematic '" + name + "'",
			Cause:   err,
		}
	}

	return buf.Bytes(), nil
}
