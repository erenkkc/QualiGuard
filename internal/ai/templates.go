package ai

type tpl struct {
	summary string
	risk    string
	example string
}

var templates = map[string]tpl{
	"python:sql-injection": {
		summary: "Kullanıcı girdisi SQL sorgusuna doğrudan ekleniyor. Parametreli sorgu kullanılmıyor.",
		risk:    "Saldırgan girdi manipülasyonu ile veritabanını okuyabilir, silebilir veya değiştirebilir.",
		example: "Girdi: ' OR '1'='1 — WHERE koşulu bypass edilir.",
	},
	"python:command-injection": {
		summary: "Dışarıdan gelen veri işletim sistemi komutuna aktarılıyor.",
		risk:    "Saldırgan sunucuda keyfi komut çalıştırabilir (RCE).",
	},
	"python:eval-usage": {
		summary: "eval() dinamik Python kodu çalıştırır; girdi kontrol altında değilse tehlikelidir.",
		risk:    "Kod enjeksiyonu ile dosya okuma veya sistem ele geçirme mümkün olabilir.",
	},
	"python:hardcoded-password": {
		summary: "Şifre, token veya API anahtarı kaynak kodda sabit yazılmış.",
		risk:    "Repo sızıntısında gizli bilgi açığa çıkar.",
	},
	"python:bare-except": {
		summary: "except: tüm hataları yakalar ve gerçek sorunları gizler.",
		risk:    "Hatalar sessizce yutulur; debug zorlaşır.",
	},
	"python:empty-except": {
		summary: "Except bloğu boş; hata işlenmiyor.",
		risk:    "Sistem hatalı durumda devam edebilir.",
	},
	"python:unused-variable": {
		summary: "Tanımlanan değişken hiç kullanılmıyor.",
		risk:    "Düşük risk — okunabilirliği düşürür.",
	},
	"python:unused-import": {
		summary: "Import edilen modül kullanılmıyor.",
		risk:    "Düşük risk — gereksiz bağımlılık algısı.",
	},
	"python:complex-function": {
		summary: "Fonksiyon karmaşıklık eşiğini aşıyor.",
		risk:    "Test etmesi zor; bug riski artar.",
	},
	"python:long-function": {
		summary: "Fonksiyon çok uzun.",
		risk:    "Bakım maliyeti artar.",
	},
	"python:syntax-error": {
		summary: "Dosyada sözdizimi hatası var.",
		risk:    "Kod çalışmaz veya deploy kırılır.",
	},
	"python:pickle-usage": {
		summary: "pickle güvenilmeyen veriyle kullanılıyor.",
		risk:    "Deserialize sırasında keyfi kod çalıştırılabilir.",
	},
	"python:weak-hash": {
		summary: "MD5 veya SHA1 kullanılıyor.",
		risk:    "Güvenlik senaryolarında yetersiz.",
	},
	"python:debug-breakpoint": {
		summary: "Debug kodu production'da kalmış.",
		risk:    "Uygulama beklenmedik şekilde durabilir.",
	},
	"python:assert-usage": {
		summary: "assert runtime doğrulama için kullanılıyor.",
		risk:    "Python -O ile assert'ler kaldırılır.",
	},
	"go:hardcoded-secret":     {summary: "Go kodunda sabit gizli bilgi var.", risk: "Credential sızıntısında açığa çıkar."},
	"go:weak-crypto":          {summary: "Zayıf hash (MD5/SHA1) kullanılıyor.", risk: "Güvenlik senaryolarında kırılabilir."},
	"go:sql-format":           {summary: "SQL fmt.Sprintf ile oluşturuluyor.", risk: "SQL injection riski."},
	"javascript:eval-usage":   {summary: "eval() ile dinamik kod çalıştırılıyor.", risk: "XSS ve kod enjeksiyonu."},
	"javascript:innerhtml-xss": {summary: "Ham HTML enjekte ediliyor.", risk: "Cross-site scripting (XSS)."},
	"javascript:hardcoded-secret": {summary: "İstemci kodunda sabit secret var.", risk: "Herkes görebilir."},
	"javascript:document-write": {summary: "document.write kullanılıyor.", risk: "DOM-based XSS riski."},
	"java:sql-concat":         {summary: "SQL string birleştirme ile yazılıyor.", risk: "PreparedStatement kullanın."},
	"java:hardcoded-secret":   {summary: "Java kodunda sabit secret var.", risk: "JAR/WAR sızıntısında okunur."},
	"java:weak-crypto":        {summary: "MD5/SHA1 kullanılıyor.", risk: "Güvenlik için yetersiz."},
	"java:script-eval":        {summary: "Dinamik script değerlendirmesi.", risk: "Kod enjeksiyonu riski."},
	"csharp:sql-concat":       {summary: "SqlCommand metni birleştirilerek yazılıyor.", risk: "Parametreli sorgu kullanın."},
	"csharp:hardcoded-secret": {summary: "C# kodunda sabit secret var.", risk: "Binary sızıntısında açığa çıkar."},
	"csharp:weak-crypto":      {summary: "MD5/SHA1 kullanılıyor.", risk: "Parola hash için zayıf."},
}
