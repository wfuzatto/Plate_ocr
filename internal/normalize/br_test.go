package normalize

import "testing"

func TestMercosul(t *testing.T) {
	c := Candidates("abc-1d23", .94, 8)
	if len(c)==0 || c[0].NormalizedText!="ABC1D23" || c[0].Format!="BR_MERCOSUL" {
		t.Fatalf("unexpected: %#v", c)
	}
}
func TestOldPlate(t *testing.T) {
	c := Candidates("ABC1234", .93, 8)
	if len(c)==0 || c[0].NormalizedText!="ABC1234" || c[0].Format!="BR_OLD" {
		t.Fatalf("unexpected: %#v", c)
	}
}
func TestAmbiguityCorrection(t *testing.T) {
	c := Candidates("ABCID23", .95, 8)
	found := false
	for _, x := range c {
		if x.NormalizedText=="ABC1D23" && x.Format=="BR_MERCOSUL" { found = true }
	}
	if !found { t.Fatalf("corrected candidate not found: %#v", c) }
}
