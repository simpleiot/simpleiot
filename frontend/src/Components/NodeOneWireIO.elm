module Components.NodeOneWireIO exposing (view)

import Api.Point as Point exposing (Point)
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Element.Input as Input
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        value =
            Point.getValue o.node.points Point.typeValue ""

        units =
            Point.getText o.node.points Point.typeUnits ""

        valueText =
            String.fromFloat (Round.roundNum 2 value)
                ++ (if units == "F" then
                        "°F"

                    else
                        "°C"
                   )

        disabled =
            Point.getBool o.node.points Point.typeDisable ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.io
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , el [ paddingXY 7 0 ] <|
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

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts ""

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts ""

                        fCheckboxInput =
                            fCheckbox opts "" Point.typeUnits "Fahrenheit?"

                        id =
                            Point.getText o.node.points Point.typeID ""
                    in
                    [ el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
                        text <|
                            "ID: "
                                ++ id
                    , textInput Point.typeDescription "Description" ""
                    , fCheckboxInput
                    , checkboxInput Point.typeDisable "Disable"
                    , counterWithReset Point.typeErrorCount Point.typeErrorCountReset "Error Count"
                    ]

                else
                    []
               )


fCheckbox :
    NodeInputs.NodeInputOptions msg
    -> String
    -> String
    -> String
    -> Element msg
fCheckbox o key typ lbl =
    Input.checkbox
        []
        { onChange =
            \d ->
                let
                    t =
                        if d then
                            "F"

                        else
                            "C"
                in
                o.onEditNodePoint
                    [ Point typ key o.now 0 0 t 0 ]
        , checked =
            Point.getText o.node.points typ key == "F"
        , icon = Input.defaultCheckbox
        , label =
            if lbl /= "" then
                Input.labelLeft [ width (px o.labelWidth) ] <|
                    el [ alignRight ] <|
                        text <|
                            lbl
                                ++ ":"

            else
                Input.labelHidden ""
        }
