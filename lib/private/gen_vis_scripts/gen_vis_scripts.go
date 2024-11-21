// Code generator for vis-encoding support scripts.
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"unicode"
)

func main() {
	for _, arg := range os.Args[1:] {
		name, dest, ok := strings.Cut(arg, "=")
		if !ok {
			log.Fatal("invalid generation spec:", arg)
		}

		f, err := os.Create(dest)
		if err != nil {
			log.Fatal(err)
		}
		defer mustClose(f)

		switch name {
		case "vis_escape_ascii.bzl":
			writeEscapeASCIIBzl(f)
		case "vis_escape_nonascii.sed":
			writeEscapeNonASCIISed(f)
		case "vis_canonicalize.sed":
			writeVisCanonicalizeSed(f)
		default:
			log.Fatal("unknown generated content:", name)
		}
	}
}

func mustClose(f *os.File) {
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

const newline rune = '\n'

// Escape all characters identified by mtree(5) as requiring escaping. Plus whitespace.
func shouldEscape(b byte) bool {
	return b == '\\' || b > unicode.MaxASCII || unicode.IsSpace(rune(b)) || !unicode.IsPrint(rune(b))
}

func writeEscapeASCIIBzl(w io.Writer) {
	fmt.Fprintln(w, strings.TrimSpace(`
# Code generated by gen_vis_scripts. DO NOT EDIT.
"A translation table for vis-encoding the ASCII range for mtree."

load(":strings.bzl", "maketrans")

VIS_ESCAPE_ASCII = maketrans({
	`))

	for i := 0; i <= unicode.MaxASCII; i++ {
		b := byte(i)
		if shouldEscape(b) {
			fmt.Fprintf(w, `    %[1]d: r"\%03[1]o",%[2]c`, b, newline)
		}
	}
	fmt.Fprintln(w, "})")
}

func writeEscapeNonASCIISed(w io.Writer) {
	fmt.Fprintln(w, strings.TrimSpace(`
# Code generated by gen_vis_scripts. DO NOT EDIT.
# Replace non-ASCII bytes with their octal escape sequences.
# Escaping of ASCII is done in Starlark prior to writing content out.
	`))
	fmt.Fprintln(w, "")

	for i := 0x80; i <= 0xFF; i++ {
		fmt.Fprintf(w, `s/\x%02[1]x/\\%03[1]o/g%[2]c`, i, newline)
	}
}

func writeVisCanonicalizeSed(w io.Writer) {
	fmt.Fprintln(w, strings.TrimSpace(`
# Code generated by gen_vis_scripts. DO NOT EDIT.
#
# Convert vis-encoded content to a bespoke canonical form. After canonicalization, equality checks are trivial.
# Backslash, space characters, and all characters outside the 95 printable ASCII set are represented using escaped three-digit octal.
# The remaining characters are not escaped; they represent themselves.
#
# Input is interpreted as libarchive would, with a wider set of escape sequences:
#   * \\, \a, \b, \f, \n, \r, \t, \v have their conventional C-based meanings
#   * \0 means NUL when not the start of an three-digit octal escape sequence
#   * \s means SPACE
#   * \ is valid as an ordinary backslash when not the start of a valid escape sequence
#
# See: https://github.com/libarchive/libarchive/blob/a90e9d84ec147be2ef6a720955f3b315cb54bca3/libarchive/archive_read_support_format_mtree.c#L1942

# Escaping of backslashes must be applied first to avoid double-interpretation.
s/\\\\|\\([^0-3abfnrstv\\]|$)/\\134\1/g
s/\\([1-3]([^0-7]|$|[0-7]([^0-7]|$)))/\\134\1/g

s/\\a/\\007/g
s/\\b/\\008/g
s/\\f/\\014/g
s/\\n/\\012/g
s/\\r/\\015/g
s/\\s/\\040/g
s/\\t/\\011/g
s/\\v/\\013/g

# NUL special form must be disambiguated from ordinary octal escape sequences.
s/\\0([^0-7]|$|[0-7]([^0-7]|$))/\\000\1/g

# Remove octal escaping from characters that don't need it.
	`))

	for i := 0; i <= 0xFF; i++ {
		b := byte(i)
		if shouldEscape(b) {
			continue
		}
		if b == '/' {
			fmt.Fprintf(w, `s:\\%03[1]o:%[1]c:g%[2]c`, b, newline)
		} else {
			fmt.Fprintf(w, `s/\\%03[1]o/%[1]c/g%[2]c`, b, newline)
		}
	}
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, "# Add octal escaping for characters that need it.")
	for i := 0; i <= 0xFF; i++ {
		b := byte(i)
		if !shouldEscape(b) {
			continue
		}
		if b == '\\' || b == '\n' {
			continue
		}
		fmt.Fprintf(w, `s/\x%02[1]x/\\%03[1]o/g%[2]c`, b, newline)
	}
}
