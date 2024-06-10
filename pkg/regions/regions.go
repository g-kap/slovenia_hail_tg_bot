package regions

func SupportedRegions() []string {
	return []string{
		"CELJE",
		"DOMŽALE",
		"IZOLA",
		"JESENICE",
		"KAMNIK",
		"KOPER",
		"KRANJ",
		"LJUBLJANA",
		"MARIBOR",
		"MURSKA SOBOTA",
		"NOVA GORICA",
		"NOVO MESTO",
		"PTUJ",
		"SLOVENJ GRADEC",
		"ŠKOFJA LOKA",
		"TRBOVLJE",
		"VELENJE",

		"BELOKRANJSKA",
		"BOVŠKA",
		"DOLENJSKA",
		"GORENJSKA",
		"GORIŠKA",
		"KOČEVSKA",
		"KOROŠKA",
		"LJUBLJANA IN OKOLICA",
		"NOTRANJSKA",
		"OBALA",
		"PODRAVJE",
		"POMURJE",
		"SAVINJSKA",
		"SPODNJE POSAVJE",
		"ZGORNJESAVSKA",
	}
}

func IsSupportedRegion(name string) bool {
	for _, r := range SupportedRegions() {
		if r == name {
			return true
		}
	}
	return false
}
