module Farmation.Is.Lcd exposing (Data, defaultData, lcd, setBlock, setPixel, setSolidBlock)

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
        _ =
            Debug.log "set block data" data

        xRange =
            List.range 0 w

        yRange =
            List.range 0 h
    in
    List.foldl
        (\y_ lcdData_ ->
            List.foldl
                (\x_ lcdData__ ->
                    let
                        v =
                            Array.get (y_ * w + x_) data
                    in
                    Array2D.set (x + x_) (y + y_) v lcdData__
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
