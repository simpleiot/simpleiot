module Components.NodeShellyIO exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        value =
            Point.getValue o.node.points Point.typeValue ""

        disabled =
            Point.getBool o.node.points Point.typeDisable ""

        typ =
            Point.getText o.node.points Point.typeType ""

        desc =
            Point.getText o.node.points Point.typeDescription ""

        summary =
            "(" ++ typ ++ ")  " ++ desc

        valueText =
            if value == 0 then
                "off"

            else
                "on"

        valueBackgroundColor =
            if valueText == "on" then
                Style.colors.blue

            else
                Style.colors.none

        valueTextColor =
            if valueText == "on" then
                Style.colors.white

            else
                Style.colors.black
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.io
            , text summary
            , el [ paddingXY 7 0, Background.color valueBackgroundColor, Font.color valueTextColor ] <|
                text <|
                    valueText
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts ""

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts ""

                        deviceID =
                            Point.getText o.node.points Point.typeDeviceID ""

                        ip =
                            Point.getText o.node.points Point.typeIP ""
                    in
                    [ textDisplay "ID" deviceID
                    , textLinkDisplay "IP" ip ("http://" ++ ip)
                    , textInput Point.typeDescription "Description" ""
                    , checkboxInput Point.typeDisable "Disable"
                    ]

                else
                    []
               )


textDisplay : String -> String -> Element msg
textDisplay label value =
    el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
        text <|
            label
                ++ ": "
                ++ value


textLinkDisplay : String -> String -> String -> Element msg
textLinkDisplay label value uri =
    el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
        row []
            [ text <|
                label
                    ++ ": "
            , newTabLink [ Font.underline ] { url = uri, label = text value }
            ]
