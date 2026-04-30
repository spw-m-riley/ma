//go:build cgo

package codectx

import (
	"strings"
	"testing"
)

func TestTrimImportsTSX(t *testing.T) {
	input := []byte(`import React, {
  useMemo,
  useState,
} from "react";
import * as path from "node:path";
import "./polyfills";
import type { Props } from "./types";

export const Widget = ({ title }: Props) => {
  return <div>{useMemo(() => title, [title])}{path.basename(title)}</div>;
};
`)

	got, warnings, err := TrimImportsFile("widget.tsx", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	for _, expected := range []string{
		"// imports: react(React, useMemo, useState); node:path(* as path); ./polyfills(side-effect)",
		"// types: Props from ./types",
		"export const Widget = ({ title }: Props) => {",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected structured imports output to include %q, got %q", expected, got)
		}
	}
	if strings.Contains(got, "import React") || strings.Contains(got, "import * as path") || strings.Contains(got, "import \"./polyfills\"") {
		t.Fatalf("expected original import lines to be removed, got %q", got)
	}
}

func TestTrimImportsJS(t *testing.T) {
	input := []byte(`import fs from "node:fs";
import * as path from "node:path";

export function summarize(file) {
  return path.basename(file) + fs.readFileSync(file, "utf8");
}
`)

	got, warnings, err := TrimImportsFile("summary.js", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if !strings.Contains(got, "// imports: node:fs(fs); node:path(* as path)") {
		t.Fatalf("expected JS import summary, got %q", got)
	}
	if strings.Contains(got, "import fs") || strings.Contains(got, "import * as path") {
		t.Fatalf("expected JS imports to be removed, got %q", got)
	}
}

func TestTrimImportsJSX(t *testing.T) {
	input := []byte(`import React from "react";
import "./polyfills";

export const Widget = ({ title }) => {
  return <div>{title}</div>;
};
`)

	got, warnings, err := TrimImportsFile("widget.jsx", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if !strings.Contains(got, "// imports: react(React); ./polyfills(side-effect)") {
		t.Fatalf("expected JSX import summary, got %q", got)
	}
	if strings.Contains(got, "import React") || strings.Contains(got, "import \"./polyfills\"") {
		t.Fatalf("expected JSX imports to be removed, got %q", got)
	}
}
