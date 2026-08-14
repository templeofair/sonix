package ollama

// Invented German letter used only in tests. Structure matches what the
// detectors care about: letterhead Datum vs a later deadline, a ß in a
// street name, and a phrase repeated three times on purpose.
const syntheticGermanPage = `# Bürgeramt Lindenstadt

Weißstr. 14 12345 Lindenstadt

Ihre Vorgangsnummer: 12 345 678 901

Datum: 10.07.2026

Erteilung einer Berechtigung zum Datenabruf Ihrer elektronischen Bescheinigungen

Sehr geehrter Herr Bergmann,

für Ihre steuerliche Identifikationsnummer ist ein Antrag zum Abruf von elektronischen Bescheinigungen eingegangen.

Möchten Sie die oben genannte Person berechtigen Ihre elektronischen Bescheinigungen abzurufen?

Möchten Sie diese Person nicht berechtigen, vernichten Sie bitte dieses Schreiben.

Möchten Sie diese Person berechtigen, prüfen Sie bitte, ob Gültigkeitsende und Veranlagungszeitraum zutreffend sind.`

const syntheticEnglishPage = `Lindenstadt Civic Office

Weißstr. 14 12345 Lindenstadt

Dear Sir or Madam, please find your invoice for the tax amount. Please check these documents as well as the deadline.`

const syntheticLetterhead = "Bürgeramt Lindenstadt\nWeißstr. 14\n\nIhre Vorgangsnummer: 12 345 678 901\nTelefon: 01234/567890\n\nDatum: 10.07.2026\n\nErteilung einer Berechtigung\nFreischaltung bis: 05.10.2026\n"
