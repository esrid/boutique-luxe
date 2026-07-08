// Package money formats integer cent amounts. Money is never a float
// anywhere in this codebase — only this package turns cents into a display
// string, and only at the edge (templates).
package money

import "fmt"

func FormatCents(cents int64) string {
	return "$" + FormatCentsPlain(cents)
}

// FormatCentsPlain is FormatCents without the currency symbol — for
// pre-filling a form input that a parser like parseDollarsToCents will
// read back, which doesn't expect a leading "$".
func FormatCentsPlain(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}
