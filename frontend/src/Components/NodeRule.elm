module Components.NodeRule exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style exposing (colors)


view : NodeOptions msg -> Element msg
view o =
    let
        opts =
            oToInputO o 100

        textInput =
            NodeInputs.nodeTextInput opts ""

        active =
            Point.getBool o.node.points "" Point.typeActive

        descBackgroundColor =
            if active then
                Style.colors.blue

            else
                Style.colors.none

        descTextColor =
            if active then
                Style.colors.white

            else
                Style.colors.black
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.list
            , el [ Background.color descBackgroundColor, Font.color descTextColor ] <|
                text <|
                    Point.getText o.node.points Point.typeDescription ""
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description" ""
                    ]

                else
                    []
               )
