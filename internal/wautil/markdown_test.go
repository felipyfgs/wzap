package wautil

import "testing"

func TestWAToMarkdown(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain", "hello world", "hello world"},
		{"bold simples", "*Certificado Digital*", "**Certificado Digital**"},
		{"bold com espaço interno", "*FIBRA X LTDA*", "**FIBRA X LTDA**"},
		{"itálico preserva underscore", "_oi_", "_oi_"},
		{"tachado", "~erro~", "~~erro~~"},
		{"misto na mesma linha", "*A* e _B_ e ~C~", "**A** e _B_ e ~~C~~"},
		{"múltiplos bolds", "*A* texto *B*", "**A** texto **B**"},
		{"asterisco isolado não converte", "2 * 3 = 6", "2 * 3 = 6"},
		{"asterisco sem fechamento", "valor *especial sem fim", "valor *especial sem fim"},
		{"espaço grudado no delimitador interno não converte", "* não negrito *", "* não negrito *"},
		{"espaço só do lado esquerdo não converte", "* esquerdo*", "* esquerdo*"},
		{"espaço só do lado direito não converte", "*direito *", "*direito *"},
		// `_X_` não é mais convertido (já é italic em ambos formatos), então
		// `snake_case_var` permanece intacto.
		{"underscore em meio de palavra preservado", "snake_case_var", "snake_case_var"},
		{"linha múltipla preserva quebra", "primeira\n*bold*\nfinal", "primeira\n**bold**\nfinal"},
		{"caso real do print", "*Certificado Digital*\n\n✅ FIBRA X LTDA *FILIAL ITINGA* -PJ A1-*R$250,00*", "**Certificado Digital**\n\n✅ FIBRA X LTDA **FILIAL ITINGA** -PJ A1-**R$250,00**"},
		// Idempotência sobre delimitadores duplos/triplos literais (caso edge:
		// alguém manda `**texto**` literal pelo WA — não devemos virar
		// `***texto***`).
		{"** literal preservado", "**Negrito**", "**Negrito**"},
		{"~~ literal preservado", "~~Tachado~~", "~~Tachado~~"},
		{"*** literal preservado", "***Bold+Italic***", "***Bold+Italic***"},
		// Misto WA + literal multi-delim.
		{"*X* + **Y** literal misturados", "*A* **B**", "**A** **B**"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WAToMarkdown(tc.in)
			if got != tc.want {
				t.Errorf("WAToMarkdown(%q) = %q, quer %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMarkdownToWA(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain", "hello world", "hello world"},
		{"bold padrão", "**negrito**", "*negrito*"},
		{"itálico em asterisco", "*itálico*", "_itálico_"},
		{"itálico em underscore mantém", "_itálico_", "_itálico_"},
		{"tachado", "~~tachado~~", "~tachado~"},
		{"código inline", "`code`", "```code```"},
		{"misto", "**B** e *I* e ~~S~~ e `c`", "*B* e _I_ e ~S~ e ```c```"},
		{"asterisco solto", "preço * 2", "preço * 2"},
		{"sem confundir bold com itálico", "**a**", "*a*"},
		{"bold seguido de italico", "**x** *y*", "*x* _y_"},
		// Conhecido: regex não suporta aninhamento; com `*` interno o `**` externo
		// não casa (mesmo comportamento do Evolution API).
		{"bold contém italico (sem aninhamento)", "**negrito com *italic* dentro**", "**negrito com _italic_ dentro**"},

		// Escapes de markdown gerados por tiptap-markdown e similares — o WA
		// não entende, removemos o backslash. Após o desescape o conteúdo
		// passa pelas conversões normais; se o operador escapou um `*` ou
		// `~` esperando literal, ele pode acabar sendo interpretado como
		// formatação WA. Trade-off aceitável dado o contexto (chat de
		// atendimento).
		{"escape em ~ vira strike WA", `\~Tachado\~`, "~Tachado~"},
		{"escape em * vira italic WA (desescape + interpretação)", `\*literal\*`, "_literal_"},
		{"escape em _ vira italic WA", `\_literal\_`, "_literal_"},
		{"escape em \\\\", `caminho \\ barra`, `caminho \ barra`},
		{"escape preserva conteúdo legítimo", `versão \(beta\)`, "versão (beta)"},

		// Hard breaks `\` no fim de linha. Removemos o backslash, mantemos o
		// newline.
		{"hard break simples", "linha1\\\nlinha2", "linha1\nlinha2"},
		{"hard break com CRLF", "linha1\\\r\nlinha2", "linha1\r\nlinha2"},
		{"hard break entre dois bolds", "**A**\\\n**B**", "*A*\n*B*"},

		// Casos compostos do print do usuário (msg 607 → msg 606 no banco
		// elodesk).
		{
			"caso real do print — Tachado escapado pelo tiptap",
			"~~Tachado~~",
			"~Tachado~",
		},
		{
			"caso real do print — bloco completo",
			"**Negrito**\n*Itálico*\n~~Tachado~~\n`code`",
			"*Negrito*\n_Itálico_\n~Tachado~\n```code```",
		},
		{
			"caso real do print — bold+strike combinado",
			"**~~Negrito e tachado~~**",
			"*~Negrito e tachado~*",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MarkdownToWA(tc.in)
			if got != tc.want {
				t.Errorf("MarkdownToWA(%q) = %q, quer %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRoundtripWA verifica que WA → MD → WA preserva a formatação para casos
// canônicos.
func TestRoundtripWA(t *testing.T) {
	cases := []string{
		"*negrito*",
		"_italico_",
		"~tachado~",
		"*A* e _B_ e ~C~",
		"FIBRA X LTDA *FILIAL ITINGA* -PJ A1-*R$250,00*",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got := MarkdownToWA(WAToMarkdown(in))
			if got != in {
				t.Errorf("roundtrip(%q) = %q", in, got)
			}
		})
	}
}
