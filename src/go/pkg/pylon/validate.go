package pylon

import (
	"fmt"
	"math"
)

// Validate walks a parsed AST and returns the complete diagnostic set.
// Emission order is stable — frontmatter first, then ref (names/refs),
// then renderer structural, then bar-data. Pure: never mutates the
// AST, never invokes a renderer.
//
// Messages are byte-identical to the JS reference implementation so
// the CI parity gate can diff them directly. Inline (SeverityWarning)
// diagnostics carry the `⚠ ` prefix to match JS's box-level rewrite
// behaviour — editors and the LSP layer can strip it if they prefer.
//
// Wire text for renderer-level diagnostics flows through
// rendererInlineError so the validator and the chart renderer cannot
// drift on wording. Any new renderer-class diagnostic must extend
// that helper rather than building a string inline.
func Validate(ast AST) []Diagnostic {
	if ast == nil {
		return nil
	}
	out := []Diagnostic{}
	out = append(out, validateFrontmatter(ast.Meta)...)
	out = append(out, validateNames(ast)...)
	out = append(out, validateRenderers(ast)...)
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
	"text":   true,
	"hbar":   true,
	"vbar":   true,
	"bar":    true,
	"banner": true,
}

// validateRenderers walks every Box (including those inside Edge
// labels) and emits every renderer-level diagnostic: unknown
// renderer, raw-string-to-non-text, bare @ref without renderer,
// missing data series, and the bar-series shape checks.
//
// The walk visits each Box once and asks rendererInlineError for the
// single diagnostic that applies. Structural errors (unknown/use-at-
// ref/bare/not-found) are collected ahead of bar-series errors so
// the stable top-level emission order (frontmatter → names →
// renderer structural → bar-data) is preserved — the parity-
// diagnostics gate depends on that ordering.
func validateRenderers(root *Box) []Diagnostic {
	structural := []Diagnostic{}
	series := []Diagnostic{}
	walkBoxes(root, func(b *Box) {
		code, msg, ok := rendererInlineError(b, root.Meta)
		if !ok {
			return
		}
		d := Diagnostic{
			Code:     code,
			Severity: SeverityWarning,
			Message:  msg,
			Span:     rendererInlineErrorSpan(b, code),
			BoxSpan:  b.Span,
		}
		if isBarSeriesCode(code) {
			series = append(series, d)
		} else {
			structural = append(structural, d)
		}
	})
	return append(structural, series...)
}

// rendererInlineError returns the diagnostic code and wire message
// for the renderer-level error that applies to b, or ok=false when
// b is renderer-clean. The message includes the leading `⚠ ` glyph
// and is byte-identical to what the JS reference emits both as a
// diagnostic and as the inline first line of a failed chart body.
//
// This helper is the single source of truth for renderer-level
// wire text: the validator wraps the result into a Diagnostic and
// the chart renderer emits the same string verbatim as the body of
// the offending box. Any future renderer or validator code that
// builds a `⚠ …` string MUST go through this helper or its
// behaviour will drift from the other surface.
//
// Bar-family diagnostics carry the user-typed renderer name
// (`bar`, `hbar`, or `vbar`) verbatim — the `bar` alias keeps its
// literal name rather than resolving to the internal `hbar`,
// mirroring what the user actually typed.
func rendererInlineError(b *Box, meta Meta) (Code, string, bool) {
	if b.Renderer == "" {
		// Bare @ref: any DataRef child in a box without a renderer is
		// a user error. Emit a single inline warning naming the first
		// DataRef; subsequent refs in the same box are silent.
		ref := firstDataRef(b)
		if ref == nil {
			return "", "", false
		}
		return CodeBareDataRef, "⚠ @" + ref.Name + ": requires | renderer", true
	}

	if !knownRenderers[b.Renderer] {
		return CodeUnknownRenderer, "⚠ unknown renderer: " + b.Renderer, true
	}

	if b.Renderer == "text" || b.Renderer == "banner" {
		// text and banner accept anything (banner silently ignores any
		// DataRef children in v1 — see PLAN.md risk #5).
		return "", "", false
	}

	// bar / hbar / vbar: require a DataRef child.
	ref := firstDataRef(b)
	if ref == nil {
		return CodeUseAtRef, "⚠ " + b.Renderer + ": use @ref", true
	}

	// DataRef present — resolve against meta.Data.
	series, ok := lookupSeries(meta.Data, ref.Name)
	if !ok {
		return CodeDataNotFound, "⚠ @" + ref.Name + " not found", true
	}

	return barSeriesInlineError(b.Renderer, series)
}

// barSeriesInlineError validates a resolved series against the
// bar-family shape rules. Returns ok=false when the series is clean;
// otherwise returns the first violation (matching JS's short-
// circuit). The name argument is the user-typed renderer name and
// threads through verbatim.
func barSeriesInlineError(name string, series interface{}) (Code, string, bool) {
	emit := func(code Code, reason string) (Code, string, bool) {
		return code, "⚠ " + name + ": " + reason, true
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
	return "", "", false
}

// rendererInlineErrorSpan picks the narrow Span for a renderer-level
// diagnostic. Bare-ref and data-not-found point at the offending
// DataRef token; every other class points at the enclosing box.
// Kept out of rendererInlineError so the helper can stay focused on
// the wire string.
func rendererInlineErrorSpan(b *Box, code Code) Span {
	if code == CodeBareDataRef || code == CodeDataNotFound {
		if ref := firstDataRef(b); ref != nil {
			return ref.Span
		}
	}
	return b.Span
}

// isBarSeriesCode reports whether code is one of the bar-series
// shape/empty/negative/duplicate classes. Used to preserve the
// top-level emission order (structural renderer errors ahead of
// bar-data errors) while still visiting each Box only once.
func isBarSeriesCode(code Code) bool {
	switch code {
	case CodeBarShape, CodeBarEmpty, CodeBarNegativeY, CodeBarDuplicateX:
		return true
	}
	return false
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
