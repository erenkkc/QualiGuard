package parser

import "github.com/qualiguard/qualiguard/internal/model"

// Analyzer parses a source file into a normalized FileAnalysis for rule engines.
type Analyzer interface {
	Language() string
	Available() bool
	AnalyzeFile(path string) (model.FileAnalysis, error)
}

type Registry struct {
	byLang map[string]Analyzer
	order  []Analyzer
}

func NewRegistry() *Registry {
	r := &Registry{byLang: make(map[string]Analyzer)}
	for _, a := range []Analyzer{
		NewPython(),
		NewGolang(),
		NewJavaScript(),
		NewJava(),
		NewCSharp(),
	} {
		r.Register(a)
	}
	return r
}

func (r *Registry) Register(a Analyzer) {
	if a == nil {
		return
	}
	lang := a.Language()
	if _, exists := r.byLang[lang]; !exists {
		r.order = append(r.order, a)
	}
	r.byLang[lang] = a
}

func (r *Registry) Get(lang string) (Analyzer, bool) {
	a, ok := r.byLang[lang]
	return a, ok
}

func (r *Registry) Languages() []string {
	out := make([]string, 0, len(r.order))
	for _, a := range r.order {
		out = append(out, a.Language())
	}
	return out
}

func (r *Registry) AnalyzeFile(lang, path string) (model.FileAnalysis, error) {
	if lang == "" {
		lang = LanguageForFilename(path)
	}
	if lang == "" {
		lang = "python"
	}
	a, ok := r.byLang[lang]
	if !ok {
		a = r.byLang["python"]
	}
	if a == nil {
		return model.FileAnalysis{}, ErrUnsupportedLanguage(lang)
	}
	if !a.Available() {
		return model.FileAnalysis{}, ErrAnalyzerUnavailable(a.Language())
	}
	return a.AnalyzeFile(path)
}

type languageError struct {
	msg string
}

func (e languageError) Error() string { return e.msg }

func ErrUnsupportedLanguage(lang string) error {
	return languageError{msg: "desteklenmeyen dil: " + lang}
}

func ErrAnalyzerUnavailable(lang string) error {
	return languageError{msg: lang + " analizi için gerekli araç bulunamadı (ör. Python 3)"}
}
