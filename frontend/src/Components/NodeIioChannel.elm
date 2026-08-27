module Components.NodeIioChannel exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


horizontalRule : Element msg
horizontalRule =
    el
        [ width fill
        , height (px 1)
        , Border.color (Element.rgb 0.5 0.5 0.5)
        , Border.widthEach { bottom = 2, top = 0, left = 0, right = 0 }
        ]
        Element.none


view : NodeOptions msg -> Element msg
view o =
    let
        value =
            Point.getValue o.node.points Point.typeValue ""

        units =
            Point.getText o.node.points Point.typeUnits ""

        valueText =
            String.fromFloat (Round.roundNum 3 value)
                ++ (if units /= "" then
                        " " ++ units

                    else
                        ""
                   )

        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        -- only an output channel accepts a requested value
        isOutput =
            Point.getText o.node.points Point.typeDirection ""
                == Point.valueOutput
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
                    ++ ": "
            , el [ paddingXY 7 0 ] <| text valueText
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            180

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        numberInput =
                            NodeInputs.nodeNumberInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts "0"

                        status label v =
                            text <| "  " ++ label ++ ": " ++ v

                        error =
                            Point.getText o.node.points Point.typeError ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeChannel "Channel" "in_voltage0"
                    , numberInput Point.typeScale "Scale"
                    , numberInput Point.typeOffset "Offset"
                    , textInput Point.typeUnits "Units" ""
                    , numberInput Point.typeMinChange "Min change"
                    , viewIf isOutput <|
                        numberInput Point.typeValueSet "Value"
                    , checkboxInput Point.typeDisabled "Disabled"
                    , horizontalRule
                    , status "Value" valueText
                    , status "Channel type" <|
                        Point.getText o.node.points Point.typeChannelType ""
                    , status "Direction" <|
                        Point.getText o.node.points Point.typeDirection ""
                    , viewIf (error /= "") <| status "Error" error
                    , counterWithReset Point.typeErrorCount
                        Point.typeErrorCountReset
                        "Error count"
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
