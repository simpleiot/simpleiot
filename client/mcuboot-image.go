package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MCUboot image layout, from bootutil/image.h. All fields are little endian.
const (
	mcubootMagic            = 0x96f3b83d
	mcubootHeaderLen        = 32
	mcubootTLVInfoMagic     = 0x6907
	mcubootTLVProtInfoMagic = 0x6908
	mcubootTLVSHA256        = 0x10
)

// mcubootMaxImageSize bounds the file read from firmwarePath. MCU flash is
// far smaller; the limit only keeps a wrong path from reading a huge file.
const mcubootMaxImageSize = 16 << 20

// mcubootVersion is an image version. The MCU reports it as
// "major.minor.revision", with ".build" appended when build is nonzero.
type mcubootVersion struct {
	major    uint8
	minor    uint8
	revision uint16
	build    uint32
}

func (v mcubootVersion) String() string {
	s := fmt.Sprintf("%v.%v.%v", v.major, v.minor, v.revision)
	if v.build != 0 {
		s += "." + strconv.FormatUint(uint64(v.build), 10)
	}
	return s
}

// compare returns -1, 0, or 1 as v is older than, equal to, or newer than o.
func (v mcubootVersion) compare(o mcubootVersion) int {
	a := []uint64{uint64(v.major), uint64(v.minor), uint64(v.revision), uint64(v.build)}
	b := []uint64{uint64(o.major), uint64(o.minor), uint64(o.revision), uint64(o.build)}
	for i := range a {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}

// parseMcubootVersion parses "M.m.r" or "M.m.r.b". A "+" before the build
// number, as imgtool writes it, is accepted too.
func parseMcubootVersion(s string) (mcubootVersion, error) {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '.' || r == '+' })
	if len(parts) != 3 && len(parts) != 4 {
		return mcubootVersion{}, fmt.Errorf("invalid version %q", s)
	}

	var n [4]uint64
	bits := []int{8, 8, 16, 32}
	for i, p := range parts {
		v, err := strconv.ParseUint(p, 10, bits[i])
		if err != nil {
			return mcubootVersion{}, fmt.Errorf("invalid version %q", s)
		}
		n[i] = v
	}

	return mcubootVersion{
		major:    uint8(n[0]),
		minor:    uint8(n[1]),
		revision: uint16(n[2]),
		build:    uint32(n[3]),
	}, nil
}

// mcubootImage is a signed image file ready to upload.
type mcubootImage struct {
	version mcubootVersion
	// hash is the SHA-256 TLV inside the image. This is what the MCU reports
	// as the image hash and what marks the image for test.
	hash []byte
	// data is the whole file, which is what gets uploaded.
	data []byte
	// sha is the SHA-256 of the whole file, sent with the first chunk.
	sha [32]byte
}

// loadMcubootImage reads and parses an image file.
func loadMcubootImage(path string) (*mcubootImage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() > mcubootMaxImageSize {
		return nil, fmt.Errorf("%v is %v bytes, over the %v byte limit",
			path, st.Size(), mcubootMaxImageSize)
	}

	b := make([]byte, st.Size())
	if _, err := f.ReadAt(b, 0); err != nil {
		return nil, err
	}
	return parseMcubootImage(b)
}

// parseMcubootImage checks the header and TLV area of an image and extracts
// its version and SHA-256 TLV. The TLV hash is also checked against the
// header, image, and protected TLVs it covers, so a truncated or corrupt
// file is refused before anything is sent to the MCU.
func parseMcubootImage(b []byte) (*mcubootImage, error) {
	if len(b) < mcubootHeaderLen {
		return nil, errors.New("image is shorter than an MCUboot header")
	}

	le := binary.LittleEndian
	if le.Uint32(b[0:]) != mcubootMagic {
		return nil, errors.New("not an MCUboot image: bad magic")
	}

	hdrSize := uint64(le.Uint16(b[8:]))
	protSize := uint64(le.Uint16(b[10:]))
	imgSize := uint64(le.Uint32(b[12:]))
	ver := mcubootVersion{
		major:    b[20],
		minor:    b[21],
		revision: le.Uint16(b[22:]),
		build:    le.Uint32(b[24:]),
	}

	if hdrSize < mcubootHeaderLen {
		return nil, fmt.Errorf("image header size %v is too small", hdrSize)
	}

	off := hdrSize + imgSize
	size := uint64(len(b))

	if protSize > 0 {
		if off+4 > size || le.Uint16(b[off:]) != mcubootTLVProtInfoMagic {
			return nil, errors.New("image is missing its protected TLV area")
		}
		if uint64(le.Uint16(b[off+2:])) != protSize {
			return nil, errors.New("image protected TLV size does not match the header")
		}
		off += protSize
	}
	hashedLen := off

	if off+4 > size || le.Uint16(b[off:]) != mcubootTLVInfoMagic {
		return nil, errors.New("image is missing its TLV area")
	}
	end := off + uint64(le.Uint16(b[off+2:]))
	if end > size {
		return nil, errors.New("image TLV area runs past the end of the file")
	}

	var hash []byte
	for p := off + 4; p < end; {
		if p+4 > end {
			return nil, errors.New("image has a truncated TLV entry")
		}
		typ := le.Uint16(b[p:])
		l := uint64(le.Uint16(b[p+2:]))
		p += 4
		if p+l > end {
			return nil, errors.New("image has a truncated TLV entry")
		}
		if typ == mcubootTLVSHA256 {
			if l != sha256.Size {
				return nil, fmt.Errorf("image SHA-256 TLV is %v bytes", l)
			}
			hash = b[p : p+l]
		}
		p += l
	}

	if hash == nil {
		return nil, errors.New("image has no SHA-256 TLV")
	}

	calc := sha256.Sum256(b[:hashedLen])
	if !bytes.Equal(calc[:], hash) {
		return nil, errors.New("image SHA-256 TLV does not match its contents")
	}

	return &mcubootImage{
		version: ver,
		hash:    append([]byte(nil), hash...),
		data:    b,
		sha:     sha256.Sum256(b),
	}, nil
}
