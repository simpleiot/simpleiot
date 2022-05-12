module Components.NodeOneWire exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        labelWidth =
            180

        opts =
            oToInputO o labelWidth

        textInput =
            NodeInputs.nodeTextInput opts ""

        numberInput =
            NodeInputs.nodeNumberInput opts ""

        counterWithReset =
            NodeInputs.nodeCounterWithReset opts ""

        checkboxInput =
            NodeInputs.nodeCheckboxInput opts ""

        disabled =
            Point.getBool o.node.points Point.typeDisable ""

        index =
            Point.getValue o.node.points Point.typeIndex ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.oneWire
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    [ el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
                        text <|
                            "Bus #: "
                                ++ String.fromFloat index
                    , textInput Point.typeDescription "Description" ""
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , checkboxInput Point.typeDisable "Disable"
                    , counterWithReset Point.typeErrorCount Point.typeErrorCountReset "Error Count"
                    ]

                else
                    []
               )
