package strcase

var uppercaseAcronym = map[string]string{
	"ID":    "id",
	"URL":   "url",
	"SKU":   "sku",
	"API":   "api",
	"JSON":  "json",
	"XML":   "xml",
	"HTML":  "html",
	"CSS":   "css",
	"SQL":   "sql",
	"HTTP":  "http",
	"HTTPS": "https",
	"FTP":   "ftp",
	"SSH":   "ssh",
	"DNS":   "dns",
	"TLS":   "tls",
	"SSL":   "ssl",
	"TCP":   "tcp",
	"UDP":   "udp",
	"IP":    "ip",
	"UI":    "ui",
	"UID":   "uid",
	"UUID":  "uuid",
	"URI":   "uri",
	"JWT":   "jwt",
	"CSV":   "csv",
	"YAML":  "yaml",
	"PDF":   "pdf",
	"RAM":   "ram",
	"CPU":   "cpu",
	"GPU":   "gpu",
	"IO":    "io",
	"DB":    "db",
	"OS":    "os",
	"SDK":   "sdk",
	"CLI":   "cli",
}

func AddAcronym(acronym, v string) {
	if _, ok := uppercaseAcronym[acronym]; ok {
		panic("acronym exists: " + acronym)
	}
	uppercaseAcronym[acronym] = v
}
