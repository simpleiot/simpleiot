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
	charMap := map[int32]uint16{46: 0x3e, 48: 0x1, 49: 0x3f, 50: 0x2, 51: 0x78, 52: 0x3, 53: 0x79, 54: 0x3c, 55: 0x7a, 56: 0x3d, 57: 0x0}
	data := []uint32{0x181f1f1f, 0x181f1f1f, 0xc1b1b1b, 0xc1b1b1b, 0xc1b1b1b, 0xc181b1b, 0x360c1b1f, 0x360c1b1f, 0x36041b18, 0x32061b18, 0x7f061b1b, 0x7f021b1b, 0x30031b1b, 0x301f1f1f, 0x301f1f1f, 0x6001f1f, 0x6001f1f, 0x7001b1b, 0x6001b1b, 0x6001b1b, 0x6001b03, 0x6000a1f, 0x6000e1f, 0x6001b1b, 0x6001b1b, 0x6001b1b, 0x6001b1b, 0x6001b1b, 0x6031f1f, 0x6031f1f, 0x3f1f1f, 0x3f1f1f, 0x33031b, 0x33031b, 0x30031b, 0x181f18, 0x181f0c, 0x18180e, 0x18181c, 0x181818, 0x81b1b, 0xc1b1b, 0xc1b1b, 0xc1f1f, 0xc1f1f}
	Font = pixfont.NewPixFont(7, 15, charMap, data)
	Font.SetVariableWidth(true)
}

