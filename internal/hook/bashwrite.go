package hook

import (
	"regexp"
	"strings"
)

var (
	// redirecciones inocuas que no escriben en el proyecto
	safeRedirRe = regexp.MustCompile(`\d*>&\d+|&>\s*/dev/null|\d*>>?\s*/dev/(null|stderr|stdout)`)
	redirRe     = regexp.MustCompile(`(?:^|[^<>&\d])\d*>>?\s*([^\s&|;<>()]+)`)
	// comandos que modifican archivos cuando están en posición de comando
	writeCmdRe = regexp.MustCompile(`(?:^|[;&|(]|\|\||&&)\s*(?:(?:sudo(?:\s+-u\s+\S+|\s+-\S+)*|command|nohup|time)\s+|env\s+(?:\S+=\S+\s+)*)*(?:tee|mv|cp|rm|touch|mkdir|rmdir|truncate|ln|dd|patch|chmod|chown|install)\b`)
	inplaceRe  = regexp.MustCompile(`\b(?:sed|perl|ruby)\s+(?:-\S+\s+)*-\S*i`)
	gitWriteRe = regexp.MustCompile(`\bgit\s+(?:-\S+\s+)*(?:(?:apply|restore|rm|mv|clean)\b|stash(?:\s+(?:push|pop|apply|drop|save|clear)\b|\s*$)|checkout\s+(?:\S+\s+)?--(?:\s|$))`)
	// find -delete/-exec y xargs que lanzan un comando que escribe
	findRe  = regexp.MustCompile(`\bfind\b[^|;&]*(?:-delete\b|-fprint\w*|-exec(?:dir)?\s+(?:\S*/)?(?:rm|mv|cp|sed|tee|chmod|chown|truncate|touch)\b)`)
	xargsRe = regexp.MustCompile(`\bxargs\b[^|;&]*\b(?:rm|mv|cp|tee|touch|chmod|chown|truncate|sed\s+-\S*i)\b`)
	// intérpretes en línea que escriben archivos o borran
	interpRe = regexp.MustCompile(`\b(?:python3?|node|ruby|perl|php)\s+(?:-\S+\s+)*-[ceE]\b.*(?:open\([^)]*['"][wax]|write_text|write_bytes|writeFile|appendFile|unlink|rmtree|os\.(?:remove|rename|system)|shutil|File\.(?:write|delete)|fs\.|subprocess|system\()`)
	// comandos que ejecutan como shell una cadena entre comillas: bash -lc '…', eval "…", ssh h "…", su -c "…"
	carrierRe = regexp.MustCompile(`\b(?:sh|bash|zsh|dash|ksh|eval|ssh|su|watch|timeout|nohup|script)\b`)
	quotedIn  = regexp.MustCompile(`'([^']*)'|"([^"]*)"`)
	quotedRe  = regexp.MustCompile(`'[^']*'|"[^"]*"`)
)

func tmpPath(p string) bool {
	p = strings.Trim(p, `"'`)
	return strings.HasPrefix(p, "/tmp/") || strings.HasPrefix(p, "/var/tmp/") || strings.HasPrefix(p, "/dev/")
}

// writesFiles indica si un comando de shell modifica archivos del proyecto: edición en
// sitio (sed -i), redirecciones, tee, mv/cp/rm…, o git que altera el árbol. Las
// escrituras en /tmp no cuentan. Es una heurística: en modo lectura es mejor un falso
// positivo (el agente puede preguntar) que dejar pasar una edición.
func writesFiles(cmd string) bool {
	if inplaceRe.MatchString(cmd) || gitWriteRe.MatchString(cmd) || findRe.MatchString(cmd) || xargsRe.MatchString(cmd) || interpRe.MatchString(cmd) {
		return true
	}
	if carrierRe.MatchString(quotedRe.ReplaceAllString(cmd, `""`)) { // el portador está fuera de las comillas
		for _, m := range quotedIn.FindAllStringSubmatch(cmd, -1) { // …y lo que ejecuta va dentro: se analiza
			if inner := m[1] + m[2]; inner != cmd && writesFiles(inner) {
				return true
			}
		}
	}
	if writeCmdRe.MatchString(cmd) {
		// permitidos si todo argumento con ruta absoluta está en /tmp y no hay rutas relativas
		if !onlyTmpArgs(cmd) {
			return true
		}
	}
	clean := quotedRe.ReplaceAllString(safeRedirRe.ReplaceAllString(cmd, " "), `""`) // un ">" dentro de comillas no es redirección
	for _, m := range redirRe.FindAllStringSubmatch(clean, -1) {
		if !tmpPath(m[1]) {
			return true
		}
	}
	return false
}

var writeWords = map[string]bool{"sudo": true, "tee": true, "mv": true, "cp": true, "rm": true, "touch": true, "mkdir": true,
	"rmdir": true, "truncate": true, "ln": true, "dd": true, "patch": true, "chmod": true, "chown": true, "install": true}

// onlyTmpArgs es cierto si todo argumento (no bandera) del comando es una ruta absoluta bajo
// /tmp (o /dev) y hay al menos uno. Un argumento relativo —p. ej. `cp /tmp/x main.go`— lo anula.
func onlyTmpArgs(cmd string) bool {
	seen := false
	for _, f := range strings.Fields(cmd) {
		switch {
		case strings.HasPrefix(f, "-"), writeWords[f], f == "&&", f == "||", f == ";", f == "|", f == "(":
			continue
		case tmpPath(f):
			seen = true
		default:
			return false
		}
	}
	return seen
}
