package pylon

import (
	"fmt"
	"math"
)

// Validate walks a parsed AST and returns the complete diagnostic set.
// Emission order is stable — frontmatter first, then ref (names/refs),
// then renderer, then bar data. Pure: never mutates the AST, never
// invokes a renderer.
//
// Messages are byte-identical to the JS reference implementation so
// the CI parity gate can diff them directly. Inline (SeverityWarning)
// diagnostics carry the `⚠ ` prefix to match JS's box-level rewrite
// behaviour — editors and the LSP layer can strip it if they prefer.
func Validate(ast AST) []Diagnostic {
	if ast == nil {
		return nil
	}
	out := []Diagnostic{}
	out = append(out, validateFrontmatter(ast.Meta)...)
	out = append(out, validateNames(ast)...)
	out = append(out, validateRenderers(ast)...)
	out = append(out, validateBarData(ast)...)
	return out
}

// validateFrontmatter converts Meta.Errors into Diagnostics. U1b's
// MetaError.Span covers the whole `data:` section for the single
// error class we emit today.
func validateFrontmatter(meta Meta) []Diagnostic {
	if len(meta.Errors) == 0 {
		return nil
	}
	out := make([]Diagnostic, 0, len(meta.Errors))
	for _, e := range meta.Errors {
		out = append(out, Diagnostic{
			Code:     CodeUnsupportedData,
			Severity: SeverityError,
			Message:  e.Message,
			Span:     e.Span,
			BoxSpan:  e.Span,
		})
	}
	return out
}

// validateNames reads the parser's side-channel accumulations from
// root (DuplicateNames / UnresolvedRefs). The span-ful records were
// captured when collectNames / resolveRefText decided to emit.
func validateNames(root *Box) []Diagnostic {
	out := []Diagnostic{}
	for _, d := range root.DuplicateNames {
		out = append(out, Diagnostic{
			Code:     CodeDuplicateName,
			Severity: SeverityError,
			Message:  "Duplicate node name: " + d.Name,
			Span:     d.Span,
			BoxSpan:  d.Span,
		})
	}
	for _, u := range root.UnresolvedRefs {
		out = append(out, Diagnostic{
			Code:     CodeUndefinedRef,
			Severity: SeverityError,
			Message:  "Undefined ref: &" + u.Name,
			Span:     u.Span,
			BoxSpan:  u.Span,
		})
	}
	return out
}

// knownRenderers mirrors the JS chartRenderers table. `bar` is a v0.1
// alias that delegates to the horizontal renderer body but threads
// its own name through for diagnostics — the user sees "⚠ bar: …"
// when they wrote `| bar`, not the internal "⚠ hbar: …".
var knownRenderers = map[string]bool{
	"text": true,
	"hbar": true,
	"vbar": true,
	"bar":  true,
}

// validateRenderers walks every Box (including those inside Edge
// labels) and emits renderer-related diagnostics: unknown renderer,
// raw-string-to-non-text, bare @ref without renderer, and missing
// data series.
func validateRenderers(root *Box) []Diagnostic {
	out := []Diagnostic{}
	walkBoxes(root, func(b *Box) {
		out = append(out, rendererDiagnosticsForBox(b, root.Meta)...)
	})
	return out
}

// rendererDiagnosticsForBox inspects a single Box for renderer-level
// issues. Flat (no recursion); walkBoxes drives the traversal.
func rendererDiagnosticsForBox(b *Box, meta Meta) []Diagnostic {
	if b.Renderer == "" {
		// Bare @ref: any DataRef child in a box without a renderer is
		// a user error. JS emits a single inline warning per box,
		// naming the first DataRef.
		ref := firstDataRef(b)
		if ref == nil {
			return nil
		}
		return []Diagnostic{{
			Code:     CodeBareDataRef,
			Severity: SeverityWarning,
			Message:  "⚠ @" + ref.Name + ": requires | renderer",
			Span:     ref.Span,
			BoxSpan:  b.Span,
		}}
	}

	if !knownRenderers[b.Renderer] {
		return []Diagnostic{{
			Code:     CodeUnknownRenderer,
			Severity: SeverityWarning,
			Message:  "⚠ unknown renderer: " + b.Renderer,
			Span:     b.Span,
			BoxSpan:  b.Span,
		}}
	}

	if b.Renderer == "text" {
		// text accepts anything; no further checks.
		return nil
	}

	// bar / hbar / vbar: require a DataRef child.
	ref := firstDataRef(b)
	if ref == nil {
		return []Diagnostic{{
			Code:     CodeUseAtRef,
			Severity: SeverityWarning,
			Message:  "⚠ " + b.Renderer + ": use @ref",
			Span:     b.Span,
			BoxSpan:  b.Span,
		}}
	}

	// DataRef present — resolve against meta.Data.
	if _, ok := lookupSeries(meta.Data, ref.Name); !ok {
		return []Diagnostic{{
			Code:     CodeDataNotFound,
			Severity: SeverityWarning,
			Message:  "⚠ @" + ref.Name + " not found",
			Span:     ref.Span,
			BoxSpan:  b.Span,
		}}
	}

	return nil
}

// validateBarData walks bar-family boxes with a resolved DataRef and
// checks the series shape / contents. Errors use the resolved
// renderer name (bar → hbar) to match JS's dispatch.
func validateBarData(root *Box) []Diagnostic {
	out := []Diagnostic{}
	walkBoxes(root, func(b *Box) {
		if b.Renderer == "" || b.Renderer == "text" {
			return
		}
		if !knownRenderers[b.Renderer] {
			return
		}
		ref := firstDataRef(b)
		if ref == nil {
			return
		}
		series, ok := lookupSeries(root.Meta.Data, ref.Name)
		if !ok {
			return
		}
		out = append(out, barSeriesDiagnostics(b, series)...)
	})
	return out
}

// barSeriesDiagnostics validates a resolved series against the
// bar-family shape rules. Emits at most one diagnostic (the first
// violation) to mirror JS's short-circuit. The renderer name threads
// through verbatim — `bar` stays `bar` in error messages to match
// what the user actually typed (JS: renderHBar(refValue, bc, budgetW,
// "bar") threads the literal name into validateBarSeries).
func barSeriesDiagnostics(b *Box, series interface{}) []Diagnostic {
	name := b.Renderer
	emit := func(code Code, msg string) []Diagnostic {
		return []Diagnostic{{
			Code:     code,
			Severity: SeverityWarning,
			Message:  "⚠ " + name + ": " + msg,
			Span:     b.Span,
			BoxSpan:  b.Span,
		}}
	}

	arr, ok := series.([]map[string]interface{})
	if !ok {
		return emit(CodeBarShape, "expected [{x,y}]")
	}
	if len(arr) == 0 {
		return emit(CodeBarEmpty, "empty series")
	}
	seen := map[string]bool{}
	for _, entry := range arr {
		if entry == nil {
			return emit(CodeBarShape, "expected [{x,y}]")
		}
		_, hasX := entry["x"]
		yRaw, hasY := entry["y"]
		if !hasX || !hasY {
			return emit(CodeBarShape, "expected [{x,y}]")
		}
		yNum, isNum := yRaw.(float64)
		if !isNum || math.IsNaN(yNum) {
			return emit(CodeBarShape, "expected [{x,y}]")
		}
		if yNum < 0 {
			return emit(CodeBarNegativeY, "negative y")
		}
		xKey := fmt.Sprintf("%v", entry["x"])
		if seen[xKey] {
			return emit(CodeBarDuplicateX, `duplicate x "`+xKey+`"`)
		}
		seen[xKey] = true
	}
	return nil
}

// firstDataRef returns the first *DataRef among a box's Items (or
// nested inside its Rows). Does NOT recurse into child Boxes — each
// Box is validated in isolation. Matches JS's filter-and-take-first.
func firstDataRef(b *Box) *DataRef {
	for _, it := range b.Items {
		switch x := it.(type) {
		case *DataRef:
			return x
		case *Row:
			for _, rit := range x.Items {
				if d, ok := rit.(*DataRef); ok {
					return d
				}
			}
		}
	}
	return nil
}

// lookupSeries resolves @name against meta.Data. Matches the JS
// contract: a flat-list dataMap only resolves `@data`; a map-keyed
// dataMap resolves whatever keys it contains. Returns the series
// value and true on success.
func lookupSeries(data interface{}, name string) (interface{}, bool) {
	if data == nil {
		return nil, false
	}
	switch d := data.(type) {
	case []map[string]interface{}:
		if name == "data" {
			return d, true
		}
		return nil, false
	case map[string]interface{}:
		v, ok := d[name]
		return v, ok
	default:
		return nil, false
	}
}

// walkBoxes visits every *Box reachable from root in pre-order,
// including those nested inside rows and edge labels. Only Boxes
// receive the callback (refs, edges, text are skipped).
func walkBoxes(n Node, fn func(*Box)) {
	if n == nil {
		return
	}
	switch x := n.(type) {
	case *Box:
		fn(x)
		for _, c := range x.Items {
			walkBoxes(c, fn)
		}
	case *Row:
		for _, c := range x.Items {
			walkBoxes(c, fn)
		}
	case *Edge:
		if x.Label != nil {
			walkBoxes(x.Label, fn)
		}
	}
}
