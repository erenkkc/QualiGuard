package rules

import "fmt"

func fixRemoveLine(current string) string {
	if current == "" {
		return "# Bu satırı silin"
	}
	return "# Sil:\n# " + current
}

func fixBareExcept(current string) string {
	if current == "" {
		return "except Exception as exc:\n    raise"
	}
	trimmed := trimSpace(current)
	if trimmed == "except:" {
		return "except Exception as exc:"
	}
	return stringsReplace(trimmed, "except:", "except Exception as exc:")
}

func fixEmptyExcept(current string) string {
	if current == "" {
		return "except Exception as exc:\n    logger.exception(\"unexpected error\")\n    raise"
	}
	return current + "\n    logger.exception(\"unexpected error\")\n    raise"
}

func fixEvalUsage(current string) string {
	if current == "" {
		return "# eval() kullanmayın — güvenli alternatif seçin"
	}
	return "# Kaldır veya güvenli parser kullan:\n# " + current
}

func fixSQLInjection(current string) string {
	_ = current
	return `cursor.execute(
    "SELECT * FROM users WHERE id = ?",
    (user_id,),
)`
}

func fixCommandInjection(current string) string {
	_ = current
	return `# Güvenli alternatif:
import subprocess
subprocess.run(["program", "arg"], check=True)
# os.system(user_input) kullanmayın`
}

func fixHardcodedSecret(name, current string) string {
	_ = current
	return fmt.Sprintf(`import os

%s = os.getenv("%s")
if not %s:
    raise RuntimeError("%s environment variable is required")`, name, name, name, name)
}

func fixUnusedVariable(name, current string) string {
	if current == "" {
		return "# Kullanılmayan '" + name + "' değişkenini silin"
	}
	return "# Sil:\n# " + current
}

func fixUnusedImport(name, current string) string {
	if current == "" {
		return "# Kullanılmayan '" + name + "' importunu silin"
	}
	return "# Sil:\n# " + current
}

func fixComplexFunction(name string) string {
	return fmt.Sprintf(`# '%s' fonksiyonunu küçük parçalara bölün:
# - yardımcı fonksiyonlar oluşturun
# - erken return kullanın
# - iç içe if'leri azaltın`, name)
}

func fixLongFunction(name string) string {
	return fmt.Sprintf(`# '%s' çok uzun — mantığı ayrı fonksiyonlara taşıyın`, name)
}

func fixPickleUsage(current string) string {
	if current == "" {
		return "# pickle yerine json veya güvenilir serialization kullanın"
	}
	return "# Kaldır veya güvenilir kaynak doğrulaması ekleyin:\n# " + current
}

func fixWeakHash(current string) string {
	_ = current
	return `import hashlib

digest = hashlib.sha256(data).hexdigest()
# Güvenlik için MD5/SHA1 yerine SHA-256 veya bcrypt kullanın`
}

func fixDebugBreakpoint(current string) string {
	if current == "" {
		return "# Debug breakpoint'i kaldırın"
	}
	return "# Sil:\n# " + current
}

func fixAssertUsage(current string) string {
	if current == "" {
		return "if not condition:\n    raise ValueError(\"validation failed\")"
	}
	return "# assert yerine explicit kontrol:\nif not (...):\n    raise ValueError(\"...\")"
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func stringsReplace(s, old, new string) string {
	if len(old) == 0 {
		return s
	}
	var out []byte
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			out = append(out, new...)
			i += len(old)
			continue
		}
		out = append(out, s[i])
		i++
	}
	return string(out)
}
