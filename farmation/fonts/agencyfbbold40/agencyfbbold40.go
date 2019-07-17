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
	charMap := map[int32]uint16{46: 0x1f2, 48: 0xf8, 49: 0xfa, 50: 0x174, 51: 0x176, 52: 0x0, 53: 0x2, 54: 0x7c, 55: 0x26c, 56: 0x7e, 57: 0x1f0}
	data := []uint32{0x7ff00e0, 0x7ff00f0, 0x7ff00f0, 0xf00f0, 0xf0070, 0xf0078, 0xf0078, 0xf0078, 0xf0078, 0xf003c, 0x3ff003c, 0x7ff003c, 0x7ff07bc, 0x7ff079e, 0x780079e, 0x780079e, 0x780079e, 0x780078f, 0x780078f, 0x780078f, 0x78f0fff, 0x78f0fff, 0x78f0fff, 0x78f0fff, 0x78f0780, 0x78f0780, 0x7ff0780, 0x7ff0780, 0x7ff0780, 0x3fe0780, 0x0, 0xfff0fff, 0xfff0fff, 0xfff0fff, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f000f, 0xf0f000f, 0xf9f000f, 0x7fe000f, 0x3fc07ff, 0x3fc0fff, 0x7fe0fff, 0xf1f0fff, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xf0f0f0f, 0xfff0fff, 0xfff0fff, 0xfff0fff, 0x7fe07fe, 0x0, 0x3c07ff, 0x3c07ff, 0x3e07ff, 0x3e0787, 0x3e0787, 0x3f0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c0787, 0x3c07ff, 0x3c07ff, 0x3c07ff, 0x3c03ff, 0x0, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf000780, 0xf0003c0, 0xfc003c0, 0x3e003c0, 0x1f001e0, 0x1f001e0, 0x7e000e0, 0xf8000f0, 0xf0000f0, 0xf000078, 0xf000078, 0xf0f0038, 0xf0f003c, 0xf0f003c, 0xf0f001e, 0xf0f001e, 0xf0f000e, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0x7fe07ff, 0x0, 0xfff, 0xfff, 0xfff, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xfff, 0xfff, 0xfff, 0xffe, 0xf00, 0xf00, 0xf00, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0x70fff, 0x70fff, 0x70fff, 0x707fe, 0x0, 0x7ff, 0x7ff, 0x7ff, 0x78f, 0x78f, 0x78f, 0x38f, 0x3c0, 0x3c0, 0x3c0, 0x3c0, 0x3c0, 0x1c0, 0x1e0, 0x1e0, 0x1e0, 0x1e0, 0xe0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0x70, 0x78, 0x78, 0x78, 0x78, 0x38, 0x3c, 0x0}
	Font = pixfont.NewPixFont(12, 31, charMap, data)
	Font.SetVariableWidth(true)
}

