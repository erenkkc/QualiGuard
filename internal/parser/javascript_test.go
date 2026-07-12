package parser

import "testing"

func TestIsDynamicInnerHTMLStatic(t *testing.T) {
	lines := []string{
		"  lightbox.innerHTML =",
		"    '<button class=\"close\">&times;</button>' +",
		"    '<img src=\"\" alt=\"\">';",
	}
	if isDynamicInnerHTML(lines, 0) {
		t.Fatal("expected static innerHTML")
	}
}

func TestIsDynamicInnerHTMLDynamic(t *testing.T) {
	lines := []string{
		"  el.innerHTML = userInput + '<span>hi</span>';",
	}
	if !isDynamicInnerHTML(lines, 0) {
		t.Fatal("expected dynamic innerHTML")
	}
}
