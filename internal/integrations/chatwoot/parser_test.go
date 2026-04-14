package chatwoot

import (
	"testing"
)

func TestFormatLocation_Full(t *testing.T) {
	locMsg := map[string]any{
		"degreesLatitude":  -23.5505,
		"degreesLongitude": -46.6333,
		"name":             "Praça da Sé",
		"address":          "São Paulo, SP",
	}
	result := formatLocation(locMsg)

	checks := []string{
		"*Localização:*",
		"_Latitude:_ -23.550500",
		"_Longitude:_ -46.633300",
		"_Nome:_ Praça da Sé",
		"_Endereço:_ São Paulo, SP",
		"_URL:_ https://www.google.com/maps/search/?api=1&query=-23.550500,-46.633300",
	}
	for _, c := range checks {
		if !containsStr(result, c) {
			t.Errorf("expected %q in result, got:\n%s", c, result)
		}
	}
}

func TestFormatLocation_NoNameNoAddress(t *testing.T) {
	locMsg := map[string]any{
		"degreesLatitude":  -15.7801,
		"degreesLongitude": -47.9292,
	}
	result := formatLocation(locMsg)

	if containsStr(result, "_Nome:_") {
		t.Error("expected no nome field in output")
	}
	if containsStr(result, "_Endereço:_") {
		t.Error("expected no endereço field in output")
	}
	if !containsStr(result, "_Latitude:_ -15.780100") {
		t.Errorf("expected latitude in result, got:\n%s", result)
	}
	if !containsStr(result, "_URL:_ https://www.google.com/maps/search/?api=1&query=-15.780100,-47.929200") {
		t.Errorf("expected URL in result, got:\n%s", result)
	}
}

func TestFormatVCard_WithPhones(t *testing.T) {
	vcard := "BEGIN:VCARD\nFN:Maria Silva\nTEL;TYPE=CELL:+5511999998888\nTEL;TYPE=HOME:+551133334444\nEND:VCARD"
	result := formatVCard(vcard)

	checks := []string{
		"*Contato:*",
		"_Nome:_ Maria Silva",
		"_Número (1):_ +5511999998888",
		"_Número (2):_ +551133334444",
	}
	for _, c := range checks {
		if !containsStr(result, c) {
			t.Errorf("expected %q in result, got:\n%s", c, result)
		}
	}
}

func TestFormatVCard_NoName(t *testing.T) {
	vcard := "BEGIN:VCARD\nTEL:+5511999998888\nEND:VCARD"
	result := formatVCard(vcard)
	if result != vcard {
		t.Errorf("expected raw vcard when no FN, got:\n%s", result)
	}
}

func TestFormatVCardWithName_OverridesFN(t *testing.T) {
	vcard := "BEGIN:VCARD\nFN:Original Name\nTEL:+5511999998888\nEND:VCARD"
	result := formatVCardWithName(vcard, "Override Name")

	if !containsStr(result, "_Nome:_ Override Name") {
		t.Errorf("expected override name, got:\n%s", result)
	}
	if containsStr(result, "Original Name") {
		t.Error("expected FN to be overridden")
	}
}

func TestFormatVCardWithName_EmptyDisplayNameUsesFN(t *testing.T) {
	vcard := "BEGIN:VCARD\nFN:John Doe\nTEL:+5511999998888\nEND:VCARD"
	result := formatVCardWithName(vcard, "")

	if !containsStr(result, "_Nome:_ John Doe") {
		t.Errorf("expected FN name, got:\n%s", result)
	}
}

func TestExtractTextFromMessage_ContactsArray_WithDisplayName(t *testing.T) {
	msg := map[string]any{
		"contactsArrayMessage": map[string]any{
			"contacts": []any{
				map[string]any{
					"displayName": "Alice Override",
					"vcard":       "BEGIN:VCARD\nFN:Alice Original\nTEL:+5511111111111\nEND:VCARD",
				},
				map[string]any{
					"displayName": "",
					"vcard":       "BEGIN:VCARD\nFN:Bob Original\nTEL:+5522222222222\nEND:VCARD",
				},
			},
		},
	}
	result := extractText(msg)

	if !containsStr(result, "_Nome:_ Alice Override") {
		t.Errorf("expected displayName override for Alice, got:\n%s", result)
	}
	if containsStr(result, "Alice Original") {
		t.Error("expected Alice's FN to be overridden by displayName")
	}
	if !containsStr(result, "_Nome:_ Bob Original") {
		t.Errorf("expected FN fallback for Bob, got:\n%s", result)
	}
}

func TestExtractTextFromMessage_ContactsArray_NoGlobalPrefix(t *testing.T) {
	msg := map[string]any{
		"contactsArrayMessage": map[string]any{
			"contacts": []any{
				map[string]any{
					"displayName": "Test",
					"vcard":       "BEGIN:VCARD\nFN:Test\nTEL:+5500000000000\nEND:VCARD",
				},
			},
		},
	}
	result := extractText(msg)

	if containsStr(result, "Contatos:") {
		t.Errorf("expected no global 'Contatos:' prefix, got:\n%s", result)
	}
	if !containsStr(result, "*Contato:*") {
		t.Errorf("expected '*Contato:*' per item, got:\n%s", result)
	}
}

func TestExtractTextFromMessage_LocationRichFormat(t *testing.T) {
	msg := map[string]any{
		"locationMessage": map[string]any{
			"degreesLatitude":  -23.5505,
			"degreesLongitude": -46.6333,
			"name":             "Test Place",
		},
	}
	result := extractText(msg)

	if !containsStr(result, "*Localização:*") {
		t.Errorf("expected rich location header, got:\n%s", result)
	}
	if !containsStr(result, "_Nome:_ Test Place") {
		t.Errorf("expected name in location, got:\n%s", result)
	}
	if !containsStr(result, "google.com/maps") {
		t.Errorf("expected Google Maps URL, got:\n%s", result)
	}
}
