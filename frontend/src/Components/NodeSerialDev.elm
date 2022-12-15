module Components.NodeSerialDev exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
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
            [ Icon.serialDev
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
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

                        log =
                            Point.getText o.node.points Point.typeLog ""

                        rate =
                            Point.getValue o.node.points Point.typeRate ""

                        rateS =
                            String.fromFloat (Round.roundNum 0 rate)
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typePort "Port" "/dev/ttyUSB0"
                    , textInput Point.typeBaud "Baud" "9600"
                    , numberInput Point.typeMaxMessageLength "Max Msg Len"
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , checkboxInput Point.typeDisable "Disable"
                    , counterWithReset Point.typeErrorCount Point.typeErrorCountReset "Error Count"
                    , counterWithReset Point.typeRx Point.typeRxReset "Rx count"
                    , counterWithReset Point.typeTx Point.typeTxReset "Tx count"
                    , text <| "  Last log: " ++ log
                    , text <| "  Rate (pts/sec): " ++ rateS
                    , viewPoints <| Point.filterSpecialPoints <| List.sortWith Point.sort o.node.points
                    ]

                else
                    []
               )


viewPoints : List Point.Point -> Element msg
viewPoints ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (Point.renderPoint >> text) ios
