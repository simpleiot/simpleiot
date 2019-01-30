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
	charMap := map[int32]uint16{46: 0x1, 48: 0x2, 54: 0x3e, 56: 0x78, 49: 0x79, 51: 0x7a, 57: 0x0, 50: 0x3, 52: 0x3c, 53: 0x3d, 55: 0x3f}
	data := []uint32{0x1f1f001f, 0x1f1f001f, 0x1b1b001b, 0x1b1b001b, 0x1b1b001b, 0x181b001b, 0xc1b001f, 0xc1b001f, 0x41b0018, 0x61b0018, 0x61b001b, 0x21b001b, 0x31b001b, 0x1f1f031f, 0x1f1f031f, 0x3f1f1f18, 0x3f1f1f18, 0x331b030c, 0x331b030c, 0x301b030c, 0x18031f0c, 0x181f1f36, 0x181f1836, 0x181b1836, 0x181b1832, 0x81b1b7f, 0xc1b1b7f, 0xc1b1b30, 0xc1f1f30, 0xc1f1f30, 0x1f061f, 0x1f061f, 0x1b071b, 0x1b061b, 0x1b061b, 0x18061b, 0xc060a, 0xe060e, 0x1c061b, 0x18061b, 0x1b061b, 0x1b061b, 0x1b061b, 0x1f061f, 0x1f061f}
	Font = pixfont.NewPixFont(7, 15, charMap, data)
	Font.SetVariableWidth(true)
}

