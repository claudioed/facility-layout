package http

import "testing"

// TestSanitizeForLog_StripsCRLF proves the CWE-117 log-injection fix: a
// crafted value containing CR/LF (the classic forged-log-line payload) has
// both stripped, so it can never be rendered as if it were a second,
// separate log line.
func TestSanitizeForLog_StripsCRLF(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no control chars", "/orders/abc-123", "/orders/abc-123"},
		{"embedded newline", "/orders/abc\n[INFO] forged line", "/orders/abc[INFO] forged line"},
		{"embedded CRLF", "/orders/abc\r\n[INFO] forged line", "/orders/abc[INFO] forged line"},
		{"embedded bare CR", "/orders/abc\rforged", "/orders/abcforged"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeForLog(tc.in); got != tc.want {
				t.Fatalf("sanitizeForLog(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
