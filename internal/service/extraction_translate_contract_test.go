package service_test

import (
	"testing"

	"github.com/templeofair/sonix/internal/ollama"
	"github.com/templeofair/sonix/internal/service"
)

func TestResolveEnglishTranslation_EmptyMustNotCopyOriginal(t *testing.T) {
	orig := "Sehr geehrte Damen und Herren, hier ist Ihre Rechnung und der Betrag ist nicht klein."
	if !ollama.LikelyNonEnglish(orig) {
		t.Fatal("fixture must look non-English")
	}
	eng, partial := service.ResolveEnglishTranslation(orig, "")
	if eng != "" {
		t.Fatalf("english = %q, want empty (must not copy original German)", eng)
	}
	if !partial {
		t.Fatal("want markPartial=true when translation empty for German source")
	}
}

func TestResolveEnglishTranslation_KeepsGoodEnglish(t *testing.T) {
	orig := "Sehr geehrte Damen und Herren, hier ist Ihre Rechnung und der Betrag ist nicht klein."
	want := "Dear Sir or Madam, here is your invoice and the amount is not small."
	eng, partial := service.ResolveEnglishTranslation(orig, want)
	if eng != want {
		t.Fatalf("english = %q, want good translation", eng)
	}
	if partial {
		t.Fatal("want markPartial=false for usable English")
	}
}

func TestResolveEnglishTranslation_EnglishSourceCopiesOriginal(t *testing.T) {
	orig := "Dear Sir or Madam, here is your invoice for the tax amount and the office."
	eng, partial := service.ResolveEnglishTranslation(orig, "")
	if eng != orig {
		t.Fatalf("english = %q, want original for English source skip", eng)
	}
	if partial {
		t.Fatal("English source must not mark partial on empty eng")
	}
}

func TestResolveEnglishTranslation_RejectsParaphrasedGerman(t *testing.T) {
	orig := "Sehr geehrte Damen und Herren, hier ist Ihre Rechnung und der Betrag ist nicht klein."
	bad := "Sehr geehrte Damen und Herren — anbei die Rechnung und der Betrag für diese Sache."
	eng, partial := service.ResolveEnglishTranslation(orig, bad)
	if eng != "" {
		t.Fatalf("english = %q, want empty for paraphrased German", eng)
	}
	if !partial {
		t.Fatal("want markPartial=true for paraphrased German")
	}
}
