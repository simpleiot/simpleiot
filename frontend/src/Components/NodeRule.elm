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
        active =
            Point.getBool o.node.points Point.typeActive ""

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

        error =
            Point.getText o.node.points Point.typeError "0"

        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        titleBackground =
            if disabled then
                Style.colors.ltgray

            else if error /= "" then
                Style.colors.red

            else
                Style.colors.none
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow
            [ spacing 10
            , paddingEach { top = 0, right = 10, bottom = 0, left = 0 }
            , Background.color titleBackground
            , width fill
            ]
            [ Icon.list
            , el [ Background.color descBackgroundColor, Font.color descTextColor ] <|
                text <|
                    Point.getText o.node.points Point.typeDescription ""
            ]
            :: (if o.expDetail then
                    let
                        opts =
                            oToInputO o 100

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"
                    in
                    [ textInput Point.typeDescription "sdfsdf" ""
                    , el [ Font.color Style.colors.red ] <| text error
                    , checkboxInput Point.typeDisabled "Disabled"
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
