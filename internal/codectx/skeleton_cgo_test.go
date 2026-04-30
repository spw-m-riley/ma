//go:build cgo

package codectx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkeletonTS(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "sample.ts")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	got, warnings, err := SkeletonFile(inputPath, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	for _, expected := range []string{
		"export function render(config: Config): string;",
		"export function save(config: Config, value: string): void;",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected structured skeleton to include %q, got %q", expected, got)
		}
	}
	if strings.Contains(got, "return readFileSync") || strings.Contains(got, "writeFileSync(destination, normalized);") {
		t.Fatalf("expected function bodies to be removed, got %q", got)
	}
}

func TestSkeletonTSX(t *testing.T) {
	input := []byte(`import React from "react";

interface Props {
  title: string;
}

type Status = "idle" | "busy";

enum Mode {
  View = "view",
}

export const Widget = ({ title }: Props) => {
  return <div>{title}</div>;
};

export class Service {
  render(value: string): string {
    return value.trim();
  }
}
`)

	got, warnings, err := SkeletonFile("widget.tsx", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	for _, expected := range []string{
		"interface Props {",
		"type Status = \"idle\" | \"busy\";",
		"enum Mode {",
		"export const Widget = ({ title }: Props) =>",
		"export class Service {",
		"render(value: string): string;",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected structured skeleton to include %q, got %q", expected, got)
		}
	}
	if strings.Contains(got, "return <div>{title}</div>;") || strings.Contains(got, "return value.trim();") {
		t.Fatalf("expected bodies to be removed, got %q", got)
	}
}

func TestSkeletonJS(t *testing.T) {
	input := []byte(`export function render(value) {
  return value.trim();
}

export class Service {
  save(value) {
    return value.toUpperCase();
  }
}
`)

	got, warnings, err := SkeletonFile("service.js", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	for _, expected := range []string{
		"export function render(value);",
		"export class Service {",
		"save(value);",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected JS skeleton to include %q, got %q", expected, got)
		}
	}
	if strings.Contains(got, "return value.trim();") || strings.Contains(got, "return value.toUpperCase();") {
		t.Fatalf("expected bodies to be removed, got %q", got)
	}
}

func TestSkeletonJSX(t *testing.T) {
	input := []byte(`import React from "react";

export const Widget = ({ title }) => {
  return <div>{title}</div>;
};
`)

	got, warnings, err := SkeletonFile("widget.jsx", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if !strings.Contains(got, "export const Widget = ({ title }) => {}") {
		t.Fatalf("expected JSX skeleton to include arrow signature, got %q", got)
	}
	if strings.Contains(got, "return <div>{title}</div>;") {
		t.Fatalf("expected JSX body to be removed, got %q", got)
	}
}

func TestSkeletonParseFallback(t *testing.T) {
	input := []byte(`export const broken = ({ value }) => {
  return value + ;
};
`)

	got, warnings, err := SkeletonFile("broken.tsx", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 2 {
		t.Fatalf("expected parse and heuristic warnings, got %#v", warnings)
	}
	if warnings[0] != "tree-sitter partial parse (ERROR nodes), using heuristic fallback" {
		t.Fatalf("unexpected parse warning %#v", warnings)
	}
	if warnings[1] != "heuristic skeleton used for non-Go source" {
		t.Fatalf("unexpected heuristic warning %#v", warnings)
	}
	if !strings.Contains(got, "export const broken = ({ value }) => {") {
		t.Fatalf("expected heuristic fallback output, got %q", got)
	}
}
