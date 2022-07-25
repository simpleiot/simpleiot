module Components.NodeSerialDev exposing (view)

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

        optionInput =
            NodeInputs.nodeOptionInput opts ""

        checkboxInput =
            NodeInputs.nodeCheckboxInput opts ""

        clientServer =
            Point.getText o.node.points Point.typeClientServer ""

        protocol =
            Point.getText o.node.points Point.typeProtocol ""

        disabled =
            Point.getBool o.node.points Point.typeDisable ""

        log =
            Point.getText o.node.points Point.typeLog ""
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
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typePort "Port" "/dev/ttyUSB0"
                    , textInput Point.typeBaud "Baud" "9600"
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , checkboxInput Point.typeDisable "Disable"
                    , counterWithReset Point.typeErrorCount Point.typeErrorCountReset "Error Count"
                    , text <| "  Last log: " ++ log
                    ]

                else
                    []
               )
