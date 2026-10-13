package client

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// buildMcubootImage builds a signed-looking MCUboot image: header, payload,
// optional protected TLVs, and unprotected TLVs holding a key hash and the
// SHA-256 TLV. It returns the file and the SHA-256 TLV value.
func buildMcubootImage(v mcubootVersion, payloadLen int, prot bool) ([]byte, []byte) {
	le := binary.LittleEndian
	const hdrSize = 64

	var protTLV []byte
	if prot {
		// one security counter entry
		protTLV = le.AppendUint16(protTLV, mcubootTLVProtInfoMagic)
		protTLV = le.AppendUint16(protTLV, 4+4+4)
		protTLV = le.AppendUint16(protTLV, 0x50)
		protTLV = le.AppendUint16(protTLV, 4)
		protTLV = le.AppendUint32(protTLV, 7)
	}

	b := make([]byte, hdrSize)
	le.PutUint32(b[0:], mcubootMagic)
	le.PutUint16(b[8:], hdrSize)
	le.PutUint16(b[10:], uint16(len(protTLV)))
	le.PutUint32(b[12:], uint32(payloadLen))
	b[20] = v.major
	b[21] = v.minor
	le.PutUint16(b[22:], v.revision)
	le.PutUint32(b[24:], v.build)

	for i := range payloadLen {
		b = append(b, byte(i*13+int(v.minor)))
	}
	b = append(b, protTLV...)

	hash := sha256.Sum256(b)

	b = le.AppendUint16(b, mcubootTLVInfoMagic)
	b = le.AppendUint16(b, 4+(4+32)+(4+32))
	b = le.AppendUint16(b, 0x01) // key hash, ignored
	b = le.AppendUint16(b, 32)
	b = append(b, make([]byte, 32)...)
	b = le.AppendUint16(b, mcubootTLVSHA256)
	b = le.AppendUint16(b, 32)
	b = append(b, hash[:]...)

	return b, hash[:]
}

func TestMcubootImageParse(t *testing.T) {
	v := mcubootVersion{major: 1, minor: 2, revision: 300, build: 4}

	for _, prot := range []bool{false, true} {
		b, hash := buildMcubootImage(v, 1000, prot)
		img, err := parseMcubootImage(b)
		if err != nil {
			t.Fatalf("prot %v: %v", prot, err)
		}
		if img.version != v {
			t.Errorf("prot %v: got version %v, expected %v", prot, img.version, v)
		}
		if string(img.hash) != string(hash) {
			t.Errorf("prot %v: wrong TLV hash", prot)
		}
		if img.sha != sha256.Sum256(b) {
			t.Errorf("prot %v: wrong file hash", prot)
		}
	}

	// a padded file, as imgtool --pad writes, still parses
	b, _ := buildMcubootImage(v, 100, true)
	if _, err := parseMcubootImage(append(b, make([]byte, 64)...)); err != nil {
		t.Errorf("trailing data: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "mcu.bin")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadMcubootImage(path); err != nil {
		t.Errorf("load: %v", err)
	}
	if _, err := loadMcubootImage(filepath.Join(dir, "missing.bin")); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestMcubootImageMalformed(t *testing.T) {
	le := binary.LittleEndian
	v := mcubootVersion{major: 1}
	good, _ := buildMcubootImage(v, 200, true)
	goodNoProt, _ := buildMcubootImage(v, 200, false)

	mod := func(src []byte, f func(b []byte) []byte) []byte {
		return f(append([]byte(nil), src...))
	}
	tlvOff := 64 + 200

	tests := []struct {
		name string
		b    []byte
	}{
		{"empty", nil},
		{"short header", good[:20]},
		{"bad magic", mod(good, func(b []byte) []byte { b[0] ^= 1; return b })},
		{"header size too small", mod(good, func(b []byte) []byte { le.PutUint16(b[8:], 16); return b })},
		{"image size past end", mod(good, func(b []byte) []byte { le.PutUint32(b[12:], 1<<20); return b })},
		{"bad protected magic", mod(good, func(b []byte) []byte { b[tlvOff] ^= 1; return b })},
		{"protected size mismatch", mod(good, func(b []byte) []byte { le.PutUint16(b[tlvOff+2:], 4); return b })},
		{"bad TLV magic", mod(goodNoProt, func(b []byte) []byte { b[tlvOff] ^= 1; return b })},
		{"TLV area past end", mod(goodNoProt, func(b []byte) []byte { le.PutUint16(b[tlvOff+2:], 1000); return b })},
		{"truncated", good[:len(good)-10]},
		{"no SHA-256 TLV", mod(goodNoProt, func(b []byte) []byte { le.PutUint16(b[tlvOff+4+36:], 0x20); return b })},
		{"wrong SHA-256 length", mod(goodNoProt, func(b []byte) []byte {
			// shorten the entry and the area total together
			le.PutUint16(b[tlvOff+2:], 4+36+4+16)
			le.PutUint16(b[tlvOff+4+36+2:], 16)
			return b[:len(b)-16]
		})},
		{"payload corrupted", mod(good, func(b []byte) []byte { b[100] ^= 1; return b })},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseMcubootImage(tc.b); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestMcubootVersion(t *testing.T) {
	parse := func(s string) mcubootVersion {
		t.Helper()
		v, err := parseMcubootVersion(s)
		if err != nil {
			t.Fatalf("%v: %v", s, err)
		}
		return v
	}

	for _, s := range []string{"1.2.3", "1.2.3.4", "0.0.0", "255.255.65535.4294967295"} {
		if got := parse(s).String(); got != s {
			t.Errorf("got %q, expected %q", got, s)
		}
	}
	if got := parse("1.2.3+4").String(); got != "1.2.3.4" {
		t.Errorf("got %q for a + build", got)
	}
	if got := parse("1.2.3.0").String(); got != "1.2.3" {
		t.Errorf("got %q, a zero build is not shown", got)
	}

	for _, s := range []string{"", "none", "1.2", "1.2.3.4.5", "256.0.0", "1.x.3", "-1.0.0"} {
		if _, err := parseMcubootVersion(s); err == nil {
			t.Errorf("%q: expected an error", s)
		}
	}

	cmp := []struct {
		a, b string
		exp  int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.10.0", "1.9.0", 1}, // numeric, not string order
		{"2.0.0", "1.255.65535.9", 1},
		{"1.0.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.0.1", -1},
		{"1.0.300", "1.0.30", 1},
	}
	for _, c := range cmp {
		if got := parse(c.a).compare(parse(c.b)); got != c.exp {
			t.Errorf("%v vs %v: got %v, expected %v", c.a, c.b, got, c.exp)
		}
	}
}
