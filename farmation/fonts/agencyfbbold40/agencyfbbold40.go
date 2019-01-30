//                                                                 XXXX     XXXXXXXXXX
//                                                                 XXXX     XXXXXXXXXX
//                                                                 XXXX     XX     XXX
//                                                                 XXXX     XX     XXX
//                                                                XXXX      XX     XXX
//                                                                XXXX      XX     XXX
//                                                                XXXX      XX     XXX
//                                                                XXXX      XX     XXX
//                                                               XXXX       XX     XXX
//                                                               XXXX       XX     XXX
//                                                               XXXX       XX     XXX
//                                                               XXXX       XX     XXX
//                                                              XXXX  XXX   XX     XXX
//                                                              XXXX  XXX   XX     XXX
//                                                              XXXX  XXX   XX     XXX
//                                                              XXXX  XXX   XX     XXX
//                                                              XXX   XXX   XX     XXX
//                                                             XXXX   XXX   XX     XXX
//                                                             XXXX   XXX   XX     XXX
//                                                             XXXX   XXX   XX     XXX
//                                                             XXXXXXXXXXXX XX     XXX
//                                                             XXXXXXXXXXXX XX     XXX
//                                                             XXXXXXXXXXXX XX     XXX
//                                                                    XXX   XX     XXX
//                                                                    XXX   XX     XXX
//                                                                    XXX   XX     XXX
//                                                                    XXX   XXXXXXXXXX
//                                                                    XXX   XXXXXXXXXX
//                                                                    XXX   XXXXXXXXX

package agencyfbbold40

import "github.com/pbnjay/pixfont"

var Font *pixfont.PixFont

func init() {
	charMap := map[int32]uint16{49: 0x1f2, 50: 0x26c, 53: 0x0, 54: 0x2, 51: 0xfa, 56: 0x1f0, 55: 0x176, 57: 0x7c, 46: 0x7e, 48: 0xf8, 52: 0x174}
	data := []uint32{0x7ff07ff, 0x7ff07ff, 0x7070007, 0x7070007, 0x7070007, 0x7070007, 0x7070007, 0x7070007, 0x70007, 0x70007, 0x703ff, 0x707ff, 0x3ff07ff, 0x7ff0700, 0x7ff0700, 0x7070700, 0x7070700, 0x7070700, 0x7070700, 0x7070700, 0x7070707, 0x7070707, 0x7070707, 0x7070707, 0x7070707, 0x7070707, 0x7ff07ff, 0x7ff07ff, 0x3fe03fe, 0x0, 0x0, 0x7ff, 0x7ff, 0x707, 0x707, 0x707, 0x707, 0x707, 0x707, 0x707, 0x707, 0x707, 0x707, 0x707, 0x7ff, 0x7ff, 0x7fe, 0x700, 0x700, 0x700, 0x700, 0x707, 0x707, 0x707, 0x707, 0x707, 0x70707, 0x707ff, 0x707ff, 0x703fe, 0x0, 0x0, 0x7ff03ff, 0x7ff03ff, 0x7070383, 0x7070383, 0x7070383, 0x7070383, 0x7070383, 0x7070383, 0x7000383, 0x7000383, 0x7800383, 0x3c00383, 0x1e00383, 0x1f00383, 0x3e00383, 0x7c00383, 0x7800383, 0x7000383, 0x7000383, 0x7000383, 0x7070383, 0x7070383, 0x7070383, 0x7070383, 0x7070383, 0x7070383, 0x7ff03ff, 0x7ff03ff, 0x3fe01ff, 0x0, 0x0, 0x7ff00f0, 0x7ff00f0, 0x78700f0, 0x78700f0, 0x3870078, 0x3c70078, 0x3c00078, 0x3c00078, 0x3c0003c, 0x3c0003c, 0x1c0003c, 0x1e0003c, 0x1e0039e, 0x1e0039e, 0x1e0039e, 0xe0039e, 0xf0038e, 0xf0038f, 0xf0038f, 0xf0038f, 0xf00fff, 0x700fff, 0x780fff, 0x780380, 0x780380, 0x780380, 0x380380, 0x3c0380, 0x3c0380, 0x0, 0x0, 0x1c0fff, 0x1c0fff, 0x1e0e07, 0x1e0e07, 0x1e0e07, 0x1f0e07, 0x1c0e07, 0x1c0e07, 0x1c0e07, 0x1c0e07, 0x1c0f9f, 0x1c07fe, 0x1c03fc, 0x1c01f8, 0x1c03fc, 0x1c0fff, 0x1c0f0f, 0x1c0e07, 0x1c0e07, 0x1c0e07, 0x1c0e07, 0x1c0e07, 0x1c0e07, 0x1c0e07, 0x1c0e07, 0x1c0e07, 0x1c0fff, 0x1c0fff, 0x1c07fe, 0x0, 0x0, 0x7ff, 0x7ff, 0x707, 0x707, 0x707, 0x707, 0x707, 0x707, 0x700, 0x780, 0x380, 0x3c0, 0x3c0, 0x1c0, 0x1e0, 0xe0, 0xf0, 0xf0, 0x78, 0x78, 0x78, 0x3c, 0x3c, 0x1e, 0x1e, 0x1f, 0x7ff, 0x7ff, 0x7ff, 0x0, 0x0}
	Font = pixfont.NewPixFont(12, 31, charMap, data)
	Font.SetVariableWidth(true)
}

