//                                                 XXXXX XXXXX
//                                                 XXXXX XXXXX
//                                                 XX XX XX XX
//                                                 XX XX XX XX
//                                                 XX XX XX XX
//                                                    XX XX XX
//                                                   XX  XX XX
//                                                   XX  XX XX
//                                                   X   XX XX
//                                                  XX   XX XX
//                                                  XX   XX XX
//                                                  X    XX XX
//                                                 XX    XX XX
//                                                 XXXXX XXXXX
//                                                 XXXXX XXXXX

package agencyfbbold20

import "github.com/pbnjay/pixfont"

var Font *pixfont.PixFont

func init() {
	charMap := map[int32]uint16{46: 0x79, 48: 0x7a, 49: 0x3d, 50: 0x3e, 51: 0x3f, 52: 0x3c, 53: 0x0, 54: 0x78, 55: 0x1, 56: 0x2, 57: 0x3}
	data := []uint32{0x1f1f3f1f, 0x1f1f3f1f, 0x1b1b3303, 0x1b1b3303, 0x1b1b3003, 0x1b1b181f, 0x1f0a181f, 0x1f0e1818, 0x181b1818, 0x181b1818, 0x1b1b081b, 0x1b1b0c1b, 0x1b1b0c1b, 0x1f1f0c1f, 0x1f1f0c1f, 0x1f1f0618, 0x1f1f0618, 0x1b1b070c, 0x1b1b060c, 0x1b1b060c, 0x1818060c, 0xc0c0636, 0xe0c0636, 0x1c040636, 0x18060632, 0x1b06067f, 0x1b02067f, 0x1b030630, 0x1f1f0630, 0x1f1f0630, 0x1f001f, 0x1f001f, 0x1b001b, 0x1b001b, 0x1b001b, 0x1b0003, 0x1b001f, 0x1b001f, 0x1b001b, 0x1b001b, 0x1b001b, 0x1b001b, 0x1b001b, 0x1f031f, 0x1f031f}
	Font = pixfont.NewPixFont(7, 15, charMap, data)
	Font.SetVariableWidth(true)
}

