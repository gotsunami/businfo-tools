package main

import "testing"

func TestSmartCapitalize(t *testing.T) {
	exps := []struct {
		city   string
		expect string
	}{
		{"grau-du-roi", "Grau-du-Roi"},
		{"saint-jean-de-védas", "Saint-Jean-de-Védas"},
		{"clermont-l'hérault", "Clermont-l'Hérault"},
		{"saint-georges-d'orques", "Saint-Georges-d'Orques"},
	}
	for _, e := range exps {
		if z := smart_capitalize(e.city); z != e.expect {
			t.Errorf("expect '%v', got '%v'", e.expect, z)
		}
	}
}
