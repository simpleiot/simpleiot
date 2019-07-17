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
	charMap := map[int32]uint16{46: 0x2, 48: 0x79, 49: 0x3, 50: 0x7a, 51: 0x3d, 52: 0x3e, 53: 0x3c, 54: 0x3f, 55: 0x0, 56: 0x1, 57: 0x78}
	data := []uint32{0x6001f3f, 0x6001f3f, 0x7001b33, 0x6001b33, 0x6001b30, 0x6001b18, 0x6000a18, 0x6000e18, 0x6001b18, 0x6001b18, 0x6001b08, 0x6001b0c, 0x6001b0c, 0x6031f0c, 0x6031f0c, 0x1f181f1f, 0x1f181f1f, 0x1b0c1b03, 0x1b0c1b03, 0x1b0c1b03, 0x30c181f, 0x1f360c1f, 0x1f360e18, 0x1b361c18, 0x1b321818, 0x1b7f1b1b, 0x1b7f1b1b, 0x1b301b1b, 0x1f301f1f, 0x1f301f1f, 0x1f1f1f, 0x1f1f1f, 0x1b1b1b, 0x1b1b1b, 0x1b1b1b, 0x181b1b, 0xc1b1f, 0xc1b1f, 0x41b18, 0x61b18, 0x61b1b, 0x21b1b, 0x31b1b, 0x1f1f1f, 0x1f1f1f}
	Font = pixfont.NewPixFont(7, 15, charMap, data)
	Font.SetVariableWidth(true)
}

