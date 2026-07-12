package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

var jsVarDecl = regexp.MustCompile(`\bvar\s+`)

func BuildFix(issue model.Issue, currentLine string) string {
	switch issue.RuleKey {
	case "python:unused-import":
		return fixUnusedImport(extractQuoted(issue.Message), currentLine)
	case "python:unused-variable":
		return fixUnusedVariable(extractQuoted(issue.Message), currentLine)
	case "python:bare-except":
		return fixBareExcept(currentLine)
	case "python:empty-except":
		return fixEmptyExcept(currentLine)
	case "python:eval-usage":
		return fixEvalUsage(currentLine)
	case "python:sql-injection":
		return fixSQLInjection(currentLine)
	case "python:command-injection":
		return fixCommandInjection(currentLine)
	case "python:hardcoded-password":
		return fixHardcodedSecret(extractQuoted(issue.Message), currentLine)
	case "python:complex-function":
		return fixComplexFunction(extractFunctionName(issue.Message))
	case "python:long-function":
		return fixLongFunction(extractFunctionName(issue.Message))
	case "python:pickle-usage":
		return fixPickleUsage(currentLine)
	case "python:weak-hash":
		return fixWeakHash(currentLine)
	case "python:debug-breakpoint":
		return fixDebugBreakpoint(currentLine)
	case "python:assert-usage":
		return fixAssertUsage(currentLine)
	case "python:syntax-error":
		return fixSyntaxError(issue.Message, currentLine)
	case "javascript:no-var":
		return fixJSNoVar(currentLine)
	case "javascript:eqeqeq":
		return fixJSEqEq(currentLine)
	case "javascript:no-console":
		return fixJSNoConsole(currentLine)
	case "javascript:innerhtml-static":
		return fixJSInnerHTMLStatic(currentLine)
	case "javascript:innerhtml-xss":
		return fixJSInnerHTMLDynamic(currentLine)
	default:
		if strings.HasPrefix(issue.RuleKey, "eslint:") {
			return fixESLintRule(strings.TrimPrefix(issue.RuleKey, "eslint:"), currentLine)
		}
		if strings.HasPrefix(issue.RuleKey, "ruff:") {
			return fixRuffRule(strings.TrimPrefix(issue.RuleKey, "ruff:"), issue.Message, currentLine)
		}
		return fixGeneric(issue)
	}
}

func fixGeneric(issue model.Issue) string {
	return fmt.Sprintf(`# Kural: %s
# Sorun: %s
#
# Ne yapın:
# 1. İlgili satırı gözden geçirin
# 2. Kural mesajındaki riski giderin
# 3. Değişiklikten sonra tekrar analiz edin`, issue.RuleKey, issue.Message)
}

func fixSyntaxError(message, currentLine string) string {
	lower := strings.ToLower(message)
	trimmed := strings.TrimSpace(currentLine)

	if looksLikeJavaScript(trimmed) {
		return fmt.Sprintf(`# Bu satır JavaScript kodu — Python analizörü parse edemez.
#
# Ne yapın:
# 1. Tam dosya için "Dosya Yükle" (.js) kullanın
# 2. Veya Canlı Analiz'de dil otomatik algılanır (sunucuyu yeniden başlatın)
#
# Hatalı (Python dosyasında):
%s
#
# JavaScript olarak doğru kullanım aynı kalır; Python dosyasına yapıştırmayın.`, displayLine(currentLine))
	}

	if strings.Contains(lower, "unexpected indent") {
		fixed := strings.TrimLeft(currentLine, " \t")
		return fmt.Sprintf(`# Girinti (indent) hatası
#
# Ne yapın:
# 1. Bu satır tek başına üst seviye kod ise baştaki boşlukları kaldırın
# 2. if/def/for içindeyse üst blokla aynı girinti seviyesinde olun
# 3. Tab yerine 4 boşluk kullanın (tab ve boşluk karışmasın)
#
# Şu an:
%s
#
# Olası düzeltme:
%s`, displayLine(currentLine), displayLine(fixed))
	}

	if strings.Contains(lower, "invalid syntax") {
		return fmt.Sprintf(`# Sözdizimi hatası
#
# Ne yapın:
# 1. Eksik parantez, tırnak veya iki nokta üst üste kontrol edin
# 2. Önceki satırın tamamlandığından emin olun
# 3. Hata satırı ve bir üst satıra birlikte bakın
#
# Hata satırı:
%s
#
# Parser mesajı: %s`, displayLine(currentLine), message)
	}

	return fmt.Sprintf(`# Sözdizimi hatası
#
# Ne yapın:
# 1. Parantez (), [], {} ve tırnak eşleşmelerini kontrol edin
# 2. Girintilerin tutarlı olduğundan emin olun
# 3. Eksik : veya , olup olmadığına bakın
#
# Satır:
%s
#
# Detay: %s`, displayLine(currentLine), message)
}

func looksLikeJavaScript(line string) bool {
	if line == "" {
		return false
	}
	markers := []string{"var ", "let ", "const ", "function ", "=>", "document.", "window.", "console."}
	for _, m := range markers {
		if strings.Contains(line, m) {
			return true
		}
	}
	return false
}

func fixJSNoVar(currentLine string) string {
	suggested := jsVarDecl.ReplaceAllString(currentLine, "const ")
	if suggested == currentLine {
		suggested = stringsReplace(currentLine, "var ", "let ")
	}
	return fmt.Sprintf(`# Ne yapın: "var" kaldırıldı — modern JavaScript'te let/const kullanın
# - Değer değişmeyecekse → const
# - Değer değişecekse → let
#
# Önce:
%s
#
# Sonra (örnek):
%s`, displayLine(currentLine), displayLine(suggested))
}

func fixJSEqEq(currentLine string) string {
	suggested := strings.ReplaceAll(strings.ReplaceAll(currentLine, "==", "==="), "!=", "!==")
	return fmt.Sprintf(`# Ne yapın: Gevşek eşitlik (==) yerine katı eşitlik (===) kullanın
# Böylece tip dönüşümü kaynaklı beklenmedik sonuçlar önlenir.
#
# Önce:
%s
#
# Sonra:
%s`, displayLine(currentLine), displayLine(suggested))
}

func fixJSNoConsole(currentLine string) string {
	return fmt.Sprintf(`# Ne yapın: Production kodunda console.log bırakmayın
# - Geliştirme loglarını kaldırın veya koşullu debug yapın
# - Gerekirse proper logger kullanın
#
# Kaldırılacak / değiştirilecek satır:
%s`, displayLine(currentLine))
}

func fixJSInnerHTMLStatic(currentLine string) string {
	return fmt.Sprintf(`# Ne yapın: Statik HTML için createElement daha güvenli ve okunaklı
#
# innerHTML yerine örnek yaklaşım:
# const box = document.createElement('div');
# box.className = 'lightbox';
# const btn = document.createElement('button');
# btn.className = 'lightbox-close';
# btn.textContent = '×';
# box.appendChild(btn);
#
# Şu anki satır:
%s`, displayLine(currentLine))
}

func fixJSInnerHTMLDynamic(currentLine string) string {
	return fmt.Sprintf(`# Ne yapın: Kullanıcı girdisi HTML'e karışmamalı
#
# 1. Mümkünse textContent kullanın (sadece metin için)
# 2. HTML şartsa güvenilir sanitize kütüphanesi kullanın
# 3. innerHTML = userInput + '...' kalıbından kaçının
#
# Riskli satır:
%s
#
# Güvenli alternatif (metin):
# element.textContent = userInput;`, displayLine(currentLine))
}

func fixRuffRule(code, message, currentLine string) string {
	switch code {
	case "F401":
		name := extractQuoted(message)
		if name == "" {
			name = "kullanılmayan_import"
		}
		return fixUnusedImport(name, currentLine)
	case "F841":
		name := extractQuoted(message)
		if name == "" {
			name = "kullanılmayan_değişken"
		}
		return fixUnusedVariable(name, currentLine)
	case "E501":
		return fmt.Sprintf(`# Ruff: Satır çok uzun (E501)
#
# Ne yapın:
# 1. Satırı mantıksal parçalara bölün
# 2. Parantez içinde çok parametre varsa her parametreyi yeni satıra alın
# 3. Uzun stringleri parçalayın veya değişkene atayın
#
# Satır:
%s`, displayLine(currentLine))
	case "I001":
		return fmt.Sprintf(`# Ruff: Import sırası düzensiz
#
# Ne yapın: importları standart kütüphane → üçüncü parti → yerel modül sırasına göre düzenleyin
#
# Satır:
%s`, displayLine(currentLine))
	default:
		if strings.HasPrefix(code, "S") {
			return fmt.Sprintf(`# Ruff güvenlik kuralı: %s
#
# Ne yapın: Güvenlik riskini giderin (sabit parola, zayıf crypto, tehlikeli API vb.)
#
# Mesaj: %s
#
# Satır:
%s`, code, message, displayLine(currentLine))
		}
		return fmt.Sprintf(`# Ruff kuralı: %s
#
# Ne yapın: Kural mesajına göre satırı düzeltin ve tekrar analiz edin
#
# Mesaj: %s
#
# Satır:
%s`, code, message, displayLine(currentLine))
	}
}

func fixESLintRule(ruleID, currentLine string) string {
	switch ruleID {
	case "no-var":
		return fixJSNoVar(currentLine)
	case "eqeqeq":
		return fixJSEqEq(currentLine)
	case "no-console":
		return fixJSNoConsole(currentLine)
	case "no-unused-vars":
		return fmt.Sprintf(`# ESLint: Kullanılmayan değişken
#
# Ne yapın:
# 1. Değişken gerçekten gereksizse satırı silin
# 2. Kullanılacaksa ilgili yerde referans verin
# 3. Bilerek bırakıyorsanız isminin başına _ ekleyin (ör. _temp)
#
# Satır:
%s`, displayLine(currentLine))
	case "prefer-const":
		return fmt.Sprintf(`# ESLint: const tercih edilir
#
# Ne yapın: Değer yeniden atanmayacaksa let yerine const yazın
#
# Satır:
%s`, displayLine(currentLine))
	default:
		return fmt.Sprintf(`# ESLint kuralı: %s
#
# Ne yapın: Kural mesajına göre satırı düzeltin ve tekrar analiz edin
#
# Satır:
%s`, ruleID, displayLine(currentLine))
	}
}

func displayLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return "# (boş satır)"
	}
	return line
}

func extractQuoted(message string) string {
	start := -1
	for i, ch := range message {
		if ch == '\'' {
			if start == -1 {
				start = i + 1
				continue
			}
			return message[start:i]
		}
	}
	return ""
}

func extractFunctionName(message string) string {
	const prefix = "Function '"
	idx := indexOf(message, prefix)
	if idx < 0 {
		return "function"
	}
	rest := message[idx+len(prefix):]
	end := indexOf(rest, "'")
	if end < 0 {
		return "function"
	}
	return rest[:end]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
