module Lcd exposing (Data, defaultData, lcd, setPixel)

import Array
import Array2D
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


setPixel : Int -> Int -> Bool -> Data -> Data
setPixel x y v data =
    Array2D.set x y v data


type alias Data =
    Array2D.Array2D Bool


defaultData : Data
defaultData =
    Array2D.repeat lcdWidth lcdHeight False


viewBoxSize =
    "0 0 "
        ++ String.fromInt svgWidth
        ++ " "
        ++ String.fromInt svgHeight


lcdDataToPixel : Int -> Int -> Bool -> Svg msg
lcdDataToPixel xPos yPos v =
    let
        xS =
            String.fromInt (xPos * lcdPxSize)

        yS =
            String.fromInt (yPos * lcdPxSize)

        fillS =
            if v then
                "black"

            else
                "none"
    in
    rect
        [ x xS
        , y yS
        , width "4"
        , height "4"
        , rx "1"
        , ry "1"
        , fill fillS
        ]
        []


lcdDataToPixels : Data -> List (Svg msg)
lcdDataToPixels data =
    List.concat
        (Array.toList
            (Array.map Array.toList
                (Array2D.indexedMap lcdDataToPixel data).data
            )
        )


lcd : Data -> Svg msg
lcd data =
    svg
        [ width (String.fromInt svgWidth)
        , height (String.fromInt svgHeight)
        , viewBox viewBoxSize
        ]
        (lcdDataToPixels data)
