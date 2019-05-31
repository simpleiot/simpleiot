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
	charMap := map[int32]uint16{46: 0x3d, 48: 0x3e, 49: 0x3f, 50: 0x0, 51: 0x78, 52: 0x79, 53: 0x1, 54: 0x2, 55: 0x7a, 56: 0x3, 57: 0x3c}
	data := []uint32{0x1f1f1f1f, 0x1f1f1f1f, 0x1b1b031b, 0x1b1b031b, 0x1b1b031b, 0x1b031f18, 0xa1f1f0c, 0xe1f180c, 0x1b1b1804, 0x1b1b1806, 0x1b1b1b06, 0x1b1b1b02, 0x1b1b1b03, 0x1f1f1f1f, 0x1f1f1f1f, 0x61f001f, 0x61f001f, 0x71b001b, 0x61b001b, 0x61b001b, 0x61b001b, 0x61b001f, 0x61b001f, 0x61b0018, 0x61b0018, 0x61b001b, 0x61b001b, 0x61b001b, 0x61f031f, 0x61f031f, 0x3f181f, 0x3f181f, 0x330c1b, 0x330c1b, 0x300c1b, 0x180c18, 0x18360c, 0x18360e, 0x18361c, 0x183218, 0x87f1b, 0xc7f1b, 0xc301b, 0xc301f, 0xc301f}
	Font = pixfont.NewPixFont(7, 15, charMap, data)
	Font.SetVariableWidth(true)
}

