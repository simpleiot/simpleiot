module Components.NodeSignalGenerator exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        value =
            Point.getValue o.node.points Point.typeValue ""

        valueText =
            String.fromFloat (Round.roundNum 2 value)

        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        summaryBackground =
            if disabled then
                Style.colors.ltgray

            else
                Style.colors.none
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10, Background.color summaryBackground ]
            [ Icon.activity
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , el [ paddingXY 7 0 ] <|
                text <|
                    valueText
                        ++ " "
                        ++ Point.getText o.node.points Point.typeUnits ""
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        numberInput =
                            NodeInputs.nodeNumberInput opts "0"

                        optionInput =
                            NodeInputs.nodeOptionInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        signalType =
                            Point.getText o.node.points Point.typeSignalType ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , checkboxInput Point.typeDisabled "Disabled"
                    , textInput Point.typeUnits "Units" ""
                    , optionInput Point.typeSignalType
                        "Signal type"
                        [ ( Point.valueSine, "Sine" )
                        , ( Point.valueSquare, "Square" )
                        , ( Point.valueTriangle, "Triangle" )
                        , ( Point.valueRandomWalk, "Random Walk" )
                        ]
                    , numberInput Point.typeMinValue "Min. Value"
                    , numberInput Point.typeMaxValue "Max. Value"
                    , numberInput Point.typeInitialValue "Initial Value"
                    , numberInput Point.typeRoundTo "Round To"
                    , numberInput Point.typeSampleRate "Sample Rate (Hz)"
                    , NodeInputs.nodeCheckboxInput opts
                        Point.keyParent
                        Point.typeDestination
                        "Sync parent node"
                    , NodeInputs.nodeCheckboxInput opts
                        Point.keyHighRate
                        Point.typeDestination
                        "High rate data"
                    , NodeInputs.nodeTextInput opts
                        Point.keyPointType
                        Point.typeDestination
                        "Point type"
                        ""
                    , NodeInputs.nodeTextInput opts
                        Point.keyPointKey
                        Point.typeDestination
                        "Point key"
                        ""
                    , numberInput Point.typeBatchPeriod "Batch Period (ms)"
                    , viewIf
                        (signalType
                            == Point.valueSine
                            || signalType
                            == Point.valueSquare
                            || signalType
                            == Point.valueTriangle
                        )
                      <|
                        numberInput Point.typeFrequency "Frequency (Hz)"
                    , viewIf (signalType == Point.valueRandomWalk) <|
                        numberInput Point.typeMinIncrement "Min. Increment"
                    , viewIf (signalType == Point.valueRandomWalk) <|
                        numberInput Point.typeMaxIncrement "Max. Increment"
                    ]

                else
                    []
               )
