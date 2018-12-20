module Lcd exposing (lcd)

import Array
import List
import Svg exposing (..)
import Svg.Attributes exposing (..)


lcdWidth =
    128


lcdHeight =
    64


lcdPxSize =
    5


svgWidth =
    lcdWidth * lcdPxSize


svgHeight =
    lcdHeight * lcdPxSize


lcdDataRow =
    Array.toList (Array.initialize lcdWidth (always False))


lcdData =
    Array.toList (Array.initialize lcdHeight (always lcdDataRow))


viewBoxSize =
    "0 0 "
        ++ String.fromInt svgWidth
        ++ " "
        ++ String.fromInt svgHeight


lcdDataToPixel : Int -> Int -> Bool -> Svg msg
lcdDataToPixel yPos xPos v =
    let
        xS =
            String.fromInt (xPos * lcdPxSize)

        yS =
            String.fromInt (yPos * lcdPxSize)
    in
    rect
        [ x xS
        , y yS
        , width "4"
        , height "4"
        , rx "1"
        , ry "1"
        ]
        []


lcdRowToPixels : Int -> List Bool -> List (Svg msg)
lcdRowToPixels yPos row =
    List.indexedMap (lcdDataToPixel yPos) row


lcd : Svg msg
lcd =
    svg
        [ width (String.fromInt svgWidth)
        , height (String.fromInt svgHeight)
        , viewBox viewBoxSize
        ]
        (List.concat
            (List.indexedMap
                lcdRowToPixels
                lcdData
            )
        )
