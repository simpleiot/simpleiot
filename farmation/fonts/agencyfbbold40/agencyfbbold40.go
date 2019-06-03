//                                                                  XXX     XXXXXXXXXXX
//                                                                 XXXX     XXXXXXXXXXX
//                                                                 XXXX     XXXXXXXXXXX
//                                                                 XXXX     XXX    XXXX
//                                                                 XXX      XXX    XXXX
//                                                                XXXX      XXX    XXXX
//                                                                XXXX      XXX    XXXX
//                                                                XXXX      XXX    XXXX
//                                                                XXXX      XXX    XXXX
//                                                               XXXX       XXX    XXXX
//                                                               XXXX       XXX    XXXX
//                                                               XXXX       XXX    XXXX
//                                                               XXXX XXXX  XXX    XXXX
//                                                              XXXX  XXXX  XXX    XXXX
//                                                              XXXX  XXXX  XXX    XXXX
//                                                              XXXX  XXXX  XXX    XXXX
//                                                              XXXX  XXXX  XXX    XXXX
//                                                             XXXX   XXXX  XXX    XXXX
//                                                             XXXX   XXXX  XXX    XXXX
//                                                             XXXX   XXXX  XXX    XXXX
//                                                             XXXXXXXXXXXX XXX    XXXX
//                                                             XXXXXXXXXXXX XXX    XXXX
//                                                             XXXXXXXXXXXX XXX    XXXX
//                                                             XXXXXXXXXXXX XXX    XXXX
//                                                                    XXXX  XXX    XXXX
//                                                                    XXXX  XXX    XXXX
//                                                                    XXXX  XXXXXXXXXXX
//                                                                    XXXX  XXXXXXXXXXX
//                                                                    XXXX  XXXXXXXXXXX
//                                                                    XXXX  XXXXXXXXXX

package agencyfbbold40

import "github.com/pbnjay/pixfont"

var Font *pixfont.PixFont

func init() {
	charMap := map[int32]uint16{46: 0x174, 48: 0x0, 49: 0x2, 50: 0x7c, 51: 0x7e, 52: 0x176, 53: 0xf8, 54: 0xfa, 55: 0x1f0, 56: 0x1f2, 57: 0x26c}
	data := []uint32{0x3c07ff, 0x3c07ff, 0x3e07ff, 0x3e0787, 0x3e0787, 0x3f0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c07ff, 0x3c07ff, 0x3c07ff, 0x3c03ff, 0x0, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf000780, 0xf0003c0, 0xfc003c0, 0x3e003c0, 0x1f001e0, 0x1f001e0, 0x7e000e0, 0xf8000f0, 0xf0000f0, 0xf000078, 0xf000078, 0xf0f0038, 0xf0f003c, 0xf0f003c, 0xf0f001e, 0xf0f001e, 0xf0f000e, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0x7fe07ff, 0x0, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0xf0f000f, 0xf0f000f, 0xf0f000f, 0xf0f000f, 0xf0f000f, 0xf0f000f, 0xf000f, 0xf03ff, 0xf07ff, 0xf07ff, 0x7ff07ff, 0xfff0780, 0xfff0780, 0xfff0780, 0xf0f0780, 0xf0f0780, 0xf0f0780, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0x7fe03fe, 0x0, 0xe00000, 0xf00000, 0xf00000, 0xf00000, 0x700000, 0x780000, 0x780000, 0x780000, 0x780000, 0x3c0000, 0x3c0000, 0x3c0000, 0x7bc0000, 0x79e0000, 0x79e0000, 0x79e0000, 0x79e0000, 0x78f0000, 0x78f0000, 0x78f0000, 0xfff0000, 0xfff0000, 0xfff0000, 0xfff0000, 0x7800000, 0x7800000, 0x7800007, 0x7800007, 0x7800007, 0x7800007, 0x0, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f038f, 0xf0f03c0, 0xf0f03c0, 0xf0f03c0, 0xf0f03c0, 0xf9f03c0, 0x7fe01c0, 0x3fc01e0, 0x3fc01e0, 0x7fe01e0, 0xf1f01e0, 0xf0f00e0, 0xf0f00f0, 0xf0f00f0, 0xf0f00f0, 0xf0f00f0, 0xf0f00f0, 0xf0f0070, 0xf0f0078, 0xf0f0078, 0xfff0078, 0xfff0078, 0xfff0038, 0x7fe003c, 0x0, 0xfff, 0xfff, 0xfff, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xfff, 0xfff, 0xfff, 0xffe, 0xf00, 0xf00, 0xf00, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xfff, 0xfff, 0xfff, 0x7fe, 0x0}
	Font = pixfont.NewPixFont(12, 31, charMap, data)
	Font.SetVariableWidth(true)
}

