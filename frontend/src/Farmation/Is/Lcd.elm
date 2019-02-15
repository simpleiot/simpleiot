module Farmation.Is.Lcd exposing (Data, Key(..), Msg(..), defaultData, lcd, setBlock, setPixel, setSolidBlock)

import Array
import Array2D
import Html
import List
import Svg exposing (..)
import Svg.Attributes exposing (..)
import Svg.Events exposing (..)


type Key
    = KeyUp
    | KeyDown
    | KeyLeft
    | KeyRight
    | KeyEnter
    | KeySK1
    | KeySK2
    | KeySK3
    | KeySK4


type Msg
    = GotKey Key


lcdWidth =
    128


lcdHeight =
    64


lcdPxSize =
    3


svgWidth =
    (lcdWidth + 2) * lcdPxSize


svgLcdHeight =
    (lcdHeight + 2) * lcdPxSize


svgHeight =
    svgLcdHeight * 3


setPixel : Int -> Int -> Bool -> Data -> Data
setPixel x y v data =
    Array2D.set x y v data


setRowSolid : Int -> Int -> Int -> Bool -> Data -> Data
setRowSolid x y w v data =
    let
        x_ =
            List.range x (x + w)

        f xSet data_ =
            Array2D.set xSet y v data_
    in
    List.foldl f data x_


setSolidBlock : Int -> Int -> Int -> Int -> Bool -> Data -> Data
setSolidBlock x y w h v data =
    let
        y_ =
            List.range y (y + h)

        f ySet data_ =
            setRowSolid x ySet w v data_
    in
    List.foldl f data y_


listToArray2D : Int -> Int -> Array.Array Bool -> Array2D.Array2D Bool
listToArray2D width height data =
    let
        data2Dinit =
            Array2D.initialize width height (\row col -> True)

        xRange =
            List.range 0 width

        yRange =
            List.range 0 height
    in
    List.foldl
        (\y data_ ->
            List.foldl
                (\x data__ ->
                    let
                        v =
                            Array.get (y * width + x) data

                        v_ =
                            case v of
                                Just v__ ->
                                    v__

                                Nothing ->
                                    False
                    in
                    Array2D.set x y v_ data__
                )
                data_
                xRange
        )
        data2Dinit
        yRange


setBlock : Int -> Int -> Int -> Int -> Array.Array Bool -> Data -> Data
setBlock x y w h data lcdData =
    let
        xRange =
            List.range 0 (w - 1)

        yRange =
            List.range 0 (h - 1)
    in
    List.foldl
        (\y_ lcdData_ ->
            List.foldl
                (\x_ lcdData__ ->
                    let
                        v =
                            Array.get (y_ * w + x_) data

                        v_ =
                            case v of
                                Just v__ ->
                                    v__

                                Nothing ->
                                    False
                    in
                    Array2D.set (x + x_) (y + y_) v_ lcdData__
                )
                lcdData_
                xRange
        )
        lcdData
        yRange


type alias Data =
    Array2D.Array2D Bool


defaultData : Data
defaultData =
    Array2D.repeat lcdWidth lcdHeight False


viewBoxSize =
    String.join " "
        [ "0 0"
        , String.fromInt svgWidth
        , String.fromInt svgHeight
        ]


lcdDataToPixel : Int -> Int -> Bool -> Svg msg
lcdDataToPixel xPos yPos v =
    let
        xS =
            String.fromInt ((xPos + 1) * lcdPxSize)

        yS =
            String.fromInt ((yPos + 1) * lcdPxSize)

        fillS =
            if v then
                "black"

            else
                "none"
    in
    rect
        [ x xS
        , y yS
        , width "3"
        , height "3"
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


button : Int -> Int -> Key -> Svg Msg
button xLoc yLoc clickMsg =
    rect
        [ x (String.fromInt xLoc)
        , y (String.fromInt yLoc)
        , rx "5"
        , width "78"
        , height "20"
        , fill "black"
        , onClick (GotKey clickMsg)
        ]
        []


arrowSize =
    50


arrowSpacing =
    10


arrow : Int -> Int -> Int -> Key -> Svg Msg
arrow xLoc yLoc rot clickMsg =
    let
        x1 =
            "0"

        y1 =
            String.fromInt (-arrowSize // 2)

        p1 =
            String.join " " [ x1, y1 ]

        x2 =
            String.fromInt (arrowSize // 2)

        y2 =
            String.fromInt (arrowSize // 2)

        p2 =
            String.join " " [ x2, y2 ]

        x3 =
            String.fromInt (-arrowSize // 2)

        y3 =
            String.fromInt (arrowSize // 2)

        p3 =
            String.join " " [ x3, y3 ]

        xRot =
            xLoc - arrowSize // 2

        yRot =
            yLoc - arrowSize // 2
    in
    polygon
        [ points <| String.join " " [ p1, p2, p3 ]
        , fill "black"
        , onClick (GotKey clickMsg)
        , transform <| String.join " " [ translate xLoc yLoc, rotate rot ]
        ]
        []


arrows : Int -> Int -> Svg Msg
arrows xLoc yLoc =
    let
        pos1 =
            arrowSize // 2

        pos2 =
            arrowSize // 2 + arrowSize + arrowSpacing

        pos3 =
            arrowSize // 2 + arrowSize * 2 + arrowSpacing * 2

        x1 =
            xLoc + pos2

        y1 =
            yLoc + pos1

        x2 =
            xLoc + pos3

        y2 =
            yLoc + pos2

        x3 =
            xLoc + pos2

        y3 =
            yLoc + pos3

        x4 =
            xLoc + pos1

        y4 =
            yLoc + pos2

        x5 =
            xLoc + pos2

        y5 =
            yLoc + pos2
    in
    g []
        [ arrow x1 y1 0 KeyUp
        , arrow x2 y2 90 KeyRight
        , arrow x3 y3 180 KeyDown
        , arrow x4 y4 -90 KeyLeft
        , enterKey x5 y5 KeyEnter
        ]


translate : Int -> Int -> String
translate x y =
    String.concat [ "translate(", String.fromInt x, " ", String.fromInt y, ")" ]


rotate : Int -> String
rotate ang =
    String.concat
        [ "rotate("
        , String.fromInt ang
        , ")"
        ]


enterKey : Int -> Int -> Key -> Svg Msg
enterKey x y clickMsg =
    circle
        [ cx "25"
        , cy "25"
        , r "25"
        , fill "black"
        , onClick (GotKey clickMsg)
        , transform <| translate (x - 25) (y - 25)
        ]
        []


lcd : Data -> Html.Html Msg
lcd data =
    svg
        [ width (String.fromInt svgWidth)
        , height (String.fromInt svgHeight)
        , viewBox viewBoxSize
        ]
        [ g []
            (lcdDataToPixels data)
        , g
            []
            [ rect
                [ x "0"
                , y "0"
                , width (String.fromInt svgWidth)
                , height (String.fromInt svgLcdHeight)
                , fill "none"
                , stroke "black"
                , strokeWidth "3"
                ]
                []
            , button 9 211 KeySK1
            , button 105 211 KeySK2
            , button 203 211 KeySK3
            , button 300 211 KeySK4
            , arrows 200 300
            ]
        ]
