package wautil

import (
	"regexp"
	"strings"
)

// Conversores entre o "markdown" do WhatsApp e o markdown padrão (CommonMark/GFM).
//
// Tabela de equivalências:
//
//	negrito:        *x* (WA) ↔ **x**  (MD)
//	itálico:        _x_ (WA) ↔ *x*    (MD, alternativa _x_ também válida em MD)
//	tachado:        ~x~ (WA) ↔ ~~x~~  (MD/GFM)
//	código inline:  `x` (WA) ↔ `x`    (MD)  — só MD→WA expande para ``` para
//	                                          preservar o comportamento do
//	                                          Evolution API.
//
// Propriedades preservadas (portadas do Evolution API mas reescritas sem
// lookbehind, que Go RE2 não suporta):
//
//   - o caractere imediatamente após o delimitador inicial e o anterior ao
//     final NÃO podem ser whitespace, evitando que `2 * 3 = 6` ou `a _ b`
//     virem formatação;
//   - o conteúdo capturado nunca atravessa newline nem o próprio delimitador;
//   - múltiplas formatações na mesma linha (`*A* e *B*`) são reconhecidas
//     independentemente.
//
// Em MarkdownToWA, os escapes de markdown gerados por editores como
// tiptap-markdown (`\~`, `\*`, `\_`, etc.) e os hard-breaks (`\` no fim da
// linha) são limpos antes de enviar ao WhatsApp; sem isso o WA renderiza o
// backslash literal e perde a formatação. Em WAToMarkdown, sequências `**`,
// `***`, `~~` e `~~~` que apareçam literais no texto vindo do WhatsApp são
// protegidas com placeholders pra que a conversão de delimitador simples
// (`*X*`) não as transforme em `***X***` por engano.

var (
	// WhatsApp → Markdown. Itálico (`_X_`) não tem regex porque é
	// idempotente entre os formatos — `_X_` é italic em CommonMark e em WA.
	waBoldRE   = regexp.MustCompile(`\*([^\s*](?:[^\n*]*?[^\s*])?)\*`)
	waStrikeRE = regexp.MustCompile(`~([^\s~](?:[^\n~]*?[^\s~])?)~`)

	// Markdown → WhatsApp.
	// Ordem de aplicação importa: tratamos delimitadores longos primeiro
	// (substituindo por placeholders NUL) pra que o passo do delimitador curto
	// não os reanalise.
	mdBoldRE       = regexp.MustCompile(`\*\*([^\s*](?:[^\n*]*?[^\s*])?)\*\*`)
	mdStrikeRE     = regexp.MustCompile(`~~([^\s~](?:[^\n~]*?[^\s~])?)~~`)
	mdInlineCodeRE = regexp.MustCompile("`([^\\s`*](?:[^\\n`*]*?[^\\s`*])?)`")
	mdItalicAstRE  = regexp.MustCompile(`\*([^\s*](?:[^\n*]*?[^\s*])?)\*`)

	// `\` no fim da linha = hard break do markdown. Removemos antes de mandar
	// pro WA (que não entende isso e renderiza o backslash literal).
	mdHardBreakRE = regexp.MustCompile(`\\(\r?\n)`)

	// Caracteres que CommonMark permite escapar com `\`. Lista canônica do
	// spec — ver https://spec.commonmark.org/0.31.2/#backslash-escapes.
	// Usamos pra desfazer o escape antes de enviar pro WA.
	mdBackslashEscapeRE = regexp.MustCompile("\\\\([!\"#$%&'()*+,\\-./:;<=>?@\\[\\\\\\]^_`{|}~])")
)

// Placeholders NUL pra proteger sequências durante as conversões. Bytes NUL
// não aparecem em texto vindo de WhatsApp/Chatwoot/elodesk (são strip-ados
// pelos transports antes de chegar aqui), então o uso é seguro.
const (
	// MarkdownToWA: marca delimitadores convertidos pra que a passagem
	// seguinte não os reanalise.
	phBold       = "\x00B"
	phStrike     = "\x00S"
	phCodeStart  = "\x00C"
	phCodeFinish = "\x00c"

	// WAToMarkdown: protege sequências literais de `**`/`~~`/`***`/`~~~`
	// vindas do WhatsApp pra que a conversão single-delim não as fragmente.
	// Ordem importa na restauração — restaurar triple antes do double.
	phLitTripleAst   = "\x00ta"
	phLitDoubleAst   = "\x00da"
	phLitTripleTilde = "\x00tt"
	phLitDoubleTilde = "\x00dt"
)

// WAToMarkdown converte texto formatado no dialeto WhatsApp para markdown
// padrão. Use só no boundary de entrada (mensagens vindas do WhatsApp).
func WAToMarkdown(s string) string {
	if s == "" {
		return s
	}
	// Proteger sequências multi-delim literais do WA antes da conversão de
	// delimitador simples. Ordem: triple antes de double, senão `***`
	// vira `**` + `*` solto.
	s = strings.ReplaceAll(s, "***", phLitTripleAst)
	s = strings.ReplaceAll(s, "**", phLitDoubleAst)
	s = strings.ReplaceAll(s, "~~~", phLitTripleTilde)
	s = strings.ReplaceAll(s, "~~", phLitDoubleTilde)

	// Usamos ${1} (e não $1) porque o Go interpreta `$1_` como nome de
	// variável `1_` em ReplaceAllString, descartando o capture.
	//
	// Italic NÃO precisa de conversão: `_X_` é italic em CommonMark e em
	// WA. Converter pra `*X*` (que era o comportamento anterior) cria
	// ambiguidade no consumidor — `*X*` poderia ser interpretado como WA
	// bold no display, virando bold por engano. Manter `_X_` é idempotente
	// nos dois formatos.
	s = waBoldRE.ReplaceAllString(s, "**${1}**")
	s = waStrikeRE.ReplaceAllString(s, "~~${1}~~")

	// Restauração inversa (triple antes de double).
	s = strings.ReplaceAll(s, phLitTripleAst, "***")
	s = strings.ReplaceAll(s, phLitDoubleAst, "**")
	s = strings.ReplaceAll(s, phLitTripleTilde, "~~~")
	s = strings.ReplaceAll(s, phLitDoubleTilde, "~~")
	return s
}

// MarkdownToWA converte markdown padrão para o dialeto WhatsApp. Use no
// boundary de saída (mensagens indo para o WhatsApp).
func MarkdownToWA(s string) string {
	if s == "" {
		return s
	}
	// Hard breaks `\` no fim da linha viram só newline. Tem que rodar antes
	// do desescape, senão o `\` que faz parte do hard break também seria
	// desescapado e bagunçaria. Após isso, qualquer `\X` restante é um
	// escape genuíno de markdown.
	s = mdHardBreakRE.ReplaceAllString(s, "${1}")

	// Desescape `\X` → `X` ANTES de aplicar as conversões. Sem isso,
	// tiptap-markdown gera `\~Tachado\~` (operador digitando `~Tachado~`
	// literal querendo testar formato WA) e o WhatsApp recebe os
	// backslashes literais — perde o strikethrough. Trade-off: se o
	// operador realmente escapou um `*` querendo asterisco literal, vai
	// virar bold/itálico no WA. Aceitável dado o contexto (Chat de
	// atendimento, não literatura).
	s = mdBackslashEscapeRE.ReplaceAllString(s, "${1}")

	s = mdBoldRE.ReplaceAllString(s, phBold+"${1}"+phBold)
	s = mdStrikeRE.ReplaceAllString(s, phStrike+"${1}"+phStrike)
	s = mdInlineCodeRE.ReplaceAllString(s, phCodeStart+"${1}"+phCodeFinish)
	s = mdItalicAstRE.ReplaceAllString(s, "_${1}_")
	s = strings.ReplaceAll(s, phBold, "*")
	s = strings.ReplaceAll(s, phStrike, "~")
	s = strings.ReplaceAll(s, phCodeStart, "```")
	s = strings.ReplaceAll(s, phCodeFinish, "```")
	return s
}
