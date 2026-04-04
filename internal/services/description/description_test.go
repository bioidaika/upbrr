// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package description

import (
	"strings"
	"testing"
)

func TestRenderBBCode(t *testing.T) {
	rendered := Render("[b]Bold[/b]\n[url=https://example.com]Link[/url]\n[list][*]One[*]Two[/list]")
	if rendered == "" {
		t.Fatalf("expected rendered output")
	}
	if !strings.Contains(rendered, "<b>Bold</b>") {
		t.Fatalf("expected bold tag, got %q", rendered)
	}
	if !strings.Contains(rendered, "<a href=\"https://example.com\">Link</a>") {
		t.Fatalf("expected link tag, got %q", rendered)
	}
	if !strings.Contains(rendered, "<ul>") || !strings.Contains(rendered, "<li>One</li>") {
		t.Fatalf("expected list tags, got %q", rendered)
	}
}

func TestRenderHTMLSanitizes(t *testing.T) {
	rendered := Render("<script>alert(1)</script><b>Safe</b>")
	if strings.Contains(rendered, "script") {
		t.Fatalf("expected script tag to be removed, got %q", rendered)
	}
	if !strings.Contains(rendered, "<b>Safe</b>") {
		t.Fatalf("expected bold tag, got %q", rendered)
	}
}

func TestRenderSanitizesURLs(t *testing.T) {
	rendered := Render("<a href=\"javascript:alert(1)\">Bad</a>")
	if strings.Contains(rendered, "javascript:") {
		t.Fatalf("expected javascript href to be removed, got %q", rendered)
	}
}

func TestRenderAllowsStyleAlignAndColor(t *testing.T) {
	rendered := Render("[center][color=red]Hi[/color][/center]")
	if !strings.Contains(rendered, "text-align: center") {
		t.Fatalf("expected text-align to be preserved, got %q", rendered)
	}
	if !strings.Contains(rendered, "color: red") {
		t.Fatalf("expected color style to be preserved, got %q", rendered)
	}
}

func TestRenderSupportsAlignEqualsBBCode(t *testing.T) {
	rendered := Render("[align=center]Hi[/align]")
	if !strings.Contains(rendered, "text-align: center") {
		t.Fatalf("expected align bbcode to render centered, got %q", rendered)
	}
	if !strings.Contains(rendered, ">Hi<") {
		t.Fatalf("expected text content preserved, got %q", rendered)
	}
}

func TestRenderPreservesSafeHTMLAlignAttribute(t *testing.T) {
	rendered := Render("<div align=\"right\">Hi</div>")
	if !strings.Contains(rendered, "align=\"right\"") {
		t.Fatalf("expected safe align attribute preserved, got %q", rendered)
	}
	if !strings.Contains(rendered, ">Hi<") {
		t.Fatalf("expected text content preserved, got %q", rendered)
	}
}

func TestRenderDoesNotDoubleEscapeHTML(t *testing.T) {
	input := "[quote]Line one\nLine two[/quote]\n[spoiler=BDInfo]Disc Title[/spoiler]"
	rendered := Render(input)
	if strings.Contains(rendered, "&lt;blockquote&gt;") {
		t.Fatalf("expected blockquote tag to be rendered, got %q", rendered)
	}
	if !strings.Contains(rendered, "<blockquote>") {
		t.Fatalf("expected blockquote tag, got %q", rendered)
	}
	if !strings.Contains(rendered, "<details>") {
		t.Fatalf("expected details tag, got %q", rendered)
	}
}

func TestRenderComparisonBBCode(t *testing.T) {
	input := "[comparison=Arrow GBR,Capelight Pictures GER]https://ptpimg.me/4p352a.png\nhttps://ptpimg.me/3bvnbe.png[/comparison]"
	rendered := Render(input)
	if !strings.Contains(rendered, "comparison__screenshots") {
		t.Fatalf("expected comparison markup, got %q", rendered)
	}
	if !strings.Contains(rendered, "comparison__image") {
		t.Fatalf("expected comparison images, got %q", rendered)
	}
	if !strings.Contains(rendered, "ptpimg.me/4p352a.png") {
		t.Fatalf("expected comparison image URL, got %q", rendered)
	}
}

func TestRenderLinkedWidthImageBBCode(t *testing.T) {
	input := "[url=https://ptpimg.me/fv71hr.png][img width=350]https://ptpimg.me/fv71hr.png[/img][/url]"
	rendered := Render(input)
	if !strings.Contains(rendered, "<a href=\"https://ptpimg.me/fv71hr.png\">") {
		t.Fatalf("expected linked image wrapper, got %q", rendered)
	}
	if !strings.Contains(rendered, "<img src=\"https://ptpimg.me/fv71hr.png\" width=\"350\"") {
		t.Fatalf("expected image width preserved, got %q", rendered)
	}
	if strings.Contains(rendered, "&lt;img") || strings.Contains(rendered, "https://ptpimg.me/fv71hr.png</a>") {
		t.Fatalf("expected no visible link text duplication, got %q", rendered)
	}
}
