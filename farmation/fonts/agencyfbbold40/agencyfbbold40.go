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
	charMap := map[int32]uint16{46: 0x2, 48: 0x7c, 49: 0x7e, 50: 0x1f0, 51: 0x1f2, 52: 0x26c, 53: 0xfa, 54: 0x174, 55: 0xf8, 56: 0x176, 57: 0x0}
	data := []uint32{0xfff, 0xfff, 0xfff, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xfff, 0xfff, 0xfff, 0xffe, 0xf00, 0xf00, 0xf00, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0x70fff, 0x70fff, 0x70fff, 0x707fe, 0x0, 0x3c07ff, 0x3c07ff, 0x3e07ff, 0x3e0787, 0x3e0787, 0x3f0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c07ff, 0x3c07ff, 0x3c07ff, 0x3c03ff, 0x0, 0x7ff07ff, 0x7ff07ff, 0x7ff07ff, 0xf078f, 0xf078f, 0xf078f, 0xf038f, 0xf03c0, 0xf03c0, 0xf03c0, 0x3ff03c0, 0x7ff03c0, 0x7ff01c0, 0x7ff01e0, 0x78001e0, 0x78001e0, 0x78001e0, 0x78000e0, 0x78000f0, 0x78000f0, 0x78f00f0, 0x78f00f0, 0x78f00f0, 0x78f0070, 0x78f0078, 0x78f0078, 0x7ff0078, 0x7ff0078, 0x7ff0038, 0x3fe003c, 0x0, 0xfff0fff, 0xfff0fff, 0xfff0fff, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f000f, 0xf0f000f, 0xf9f000f, 0x7fe000f, 0x3fc07ff, 0x3fc0fff, 0x7fe0fff, 0xf1f0fff, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xfff0fff, 0xfff0fff, 0xfff0fff, 0x7fe07fe, 0x0, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf000780, 0xf0003c0, 0xfc003c0, 0x3e003c0, 0x1f001e0, 0x1f001e0, 0x7e000e0, 0xf8000f0, 0xf0000f0, 0xf000078, 0xf000078, 0xf0f0038, 0xf0f003c, 0xf0f003c, 0xf0f001e, 0xf0f001e, 0xf0f000e, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0x7fe07ff, 0x0, 0xe0, 0xf0, 0xf0, 0xf0, 0x70, 0x78, 0x78, 0x78, 0x78, 0x3c, 0x3c, 0x3c, 0x7bc, 0x79e, 0x79e, 0x79e, 0x79e, 0x78f, 0x78f, 0x78f, 0xfff, 0xfff, 0xfff, 0xfff, 0x780, 0x780, 0x780, 0x780, 0x780, 0x780, 0x0}
	Font = pixfont.NewPixFont(12, 31, charMap, data)
	Font.SetVariableWidth(true)
}

