package shacl

import (
	"encoding/json"
	"math/big"
	"regexp"
	"strconv"
)

// wellFormed reports whether a literal's lexical form is in the lexical space
// of its datatype, for the datatypes of the D8 table (#14). SHACL §4.1.2 makes
// an ill-typed literal a sh:datatype violation. The validator keeps the term
// export writes (design D9), and export can write an ill-typed form: a
// dateTime whose timezone offset is beyond ±14:00, or a string holding a
// character outside XML's Char. A literal of any other datatype is reported
// well formed: no sh:datatype in the profile can name it.
func wellFormed(t term) bool {
	switch t.datatype {
	case xsdString:
		return isXMLChars(t.value)
	case xsdInt:
		_, ok := parseXSDInteger(t.value)
		return ok
	case xsdDouble:
		_, _, ok := parseXSDDouble(t.value)
		return ok
	case xsdBool:
		return t.value == "true" || t.value == "false" || t.value == "1" || t.value == "0"
	case xsdDateTime:
		return isXSDDateTime(t.value)
	case rdfJSON:
		return json.Valid([]byte(t.value))
	}
	return true
}

// isXMLChars is xsd:string's lexical space: XSD 1.1 Part 2 §3.3.1 takes the
// Char production from XML, and lets a processor choose XML 1.0 or 1.1; this
// uses XML 1.0 (§2.2), #x9 | #xA | #xD | [#x20-#xD7FF] | [#xE000-#xFFFD] |
// [#x10000-#x10FFFF], the stricter of the two. The string is valid UTF-8
// (ruling Q31), so it holds no surrogate.
func isXMLChars(s string) bool {
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0xFFFE || r == 0xFFFF {
			return false
		}
	}
	return true
}

// xsdDateTimeLexical is dateTimeLexicalRep of XSD 1.1 Part 2 §3.3.7.2: year,
// month, day, time or end of day, and an optional timezone of at most ±14:00.
var xsdDateTimeLexical = regexp.MustCompile(`^(-?(?:[1-9][0-9]{3,}|0[0-9]{3}))-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])` +
	`T(?:(?:[01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](?:\.[0-9]+)?|24:00:00(?:\.0+)?)` +
	`(?:Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`)

// isXSDDateTime checks the lexical form and the day-of-month constraint
// (§3.3.7, after the proleptic Gregorian calendar, in which year 0000 is a
// leap year).
func isXSDDateTime(s string) bool {
	m := xsdDateTimeLexical.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	switch month {
	case 4, 6, 9, 11:
		return day <= 30
	case 2:
		year, _ := new(big.Int).SetString(m[1], 10)
		mod := func(n int64) int64 { return new(big.Int).Mod(year, big.NewInt(n)).Int64() }
		leap := mod(400) == 0 || mod(4) == 0 && mod(100) != 0
		return day <= 28 || day == 29 && leap
	}
	return true
}
