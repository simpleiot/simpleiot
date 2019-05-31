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
	charMap := map[int32]uint16{46: 0xf8, 48: 0xfa, 49: 0x0, 50: 0x2, 51: 0x26c, 52: 0x7c, 53: 0x174, 54: 0x176, 55: 0x1f0, 56: 0x1f2, 57: 0x7e}
	data := []uint32{0x7ff003c, 0x7ff003c, 0x7ff003e, 0x78f003e, 0x78f003e, 0x78f003f, 0x78f003c, 0x78f003c, 0x78f003c, 0x780003c, 0x3c0003c, 0x3c0003c, 0x3c0003c, 0x1e0003c, 0x1e0003c, 0xe0003c, 0xf0003c, 0xf0003c, 0x78003c, 0x78003c, 0x38003c, 0x3c003c, 0x3c003c, 0x1e003c, 0x1e003c, 0xe003c, 0x7ff003c, 0x7ff003c, 0x7ff003c, 0x7ff003c, 0x0, 0xfff00e0, 0xfff00f0, 0xfff00f0, 0xf0f00f0, 0xf0f0070, 0xf0f0078, 0xf0f0078, 0xf0f0078, 0xf0f0078, 0xf0f003c, 0xf0f003c, 0xf0f003c, 0xf0f07bc, 0xfff079e, 0xfff079e, 0xfff079e, 0xffe079e, 0xf00078f, 0xf00078f, 0xf00078f, 0xf0f0fff, 0xf0f0fff, 0xf0f0fff, 0xf0f0fff, 0xf0f0780, 0xf0f0780, 0xfff0780, 0xfff0780, 0xfff0780, 0x7fe0780, 0x0, 0x7ff0000, 0x7ff0000, 0x7ff0000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7870000, 0x7ff0007, 0x7ff0007, 0x7ff0007, 0x3ff0007, 0x0, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0xf0f000f, 0xf0f000f, 0xf0f000f, 0xf0f000f, 0xf0f000f, 0xf0f000f, 0xf000f, 0xf03ff, 0xf07ff, 0xf07ff, 0x7ff07ff, 0xfff0780, 0xfff0780, 0xfff0780, 0xf0f0780, 0xf0f0780, 0xf0f0780, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0x7fe03fe, 0x0, 0xfff07ff, 0xfff07ff, 0xfff07ff, 0xf0f078f, 0xf0f078f, 0xf0f078f, 0xf0f038f, 0xf0f03c0, 0xf0f03c0, 0xf0f03c0, 0xf0f03c0, 0xf9f03c0, 0x7fe01c0, 0x3fc01e0, 0x3fc01e0, 0x7fe01e0, 0xf1f01e0, 0xf0f00e0, 0xf0f00f0, 0xf0f00f0, 0xf0f00f0, 0xf0f00f0, 0xf0f00f0, 0xf0f0070, 0xf0f0078, 0xf0f0078, 0xfff0078, 0xfff0078, 0xfff0038, 0x7fe003c, 0x0, 0xfff, 0xfff, 0xfff, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf00, 0xf00, 0xfc0, 0x3e0, 0x1f0, 0x1f0, 0x7e0, 0xf80, 0xf00, 0xf00, 0xf00, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xf0f, 0xfff, 0xfff, 0xfff, 0x7fe, 0x0}
	Font = pixfont.NewPixFont(12, 31, charMap, data)
	Font.SetVariableWidth(true)
}

