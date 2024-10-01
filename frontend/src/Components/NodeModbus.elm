module Components.NodeModbus exposing (view)

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
        disabled =
            Point.getBool o.node.points Point.typeDisabled ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.bus
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
                            NodeInputs.nodeTextInput opts "0"

                        numberInput =
                            NodeInputs.nodeNumberInput opts "0"

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts "0"

                        optionInput =
                            NodeInputs.nodeOptionInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        clientServer =
                            Point.getText o.node.points Point.typeClientServer ""

                        protocol =
                            Point.getText o.node.points Point.typeProtocol ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , optionInput Point.typeClientServer
                        "Client/Server"
                        [ ( Point.valueClient, "client" )
                        , ( Point.valueServer, "server" )
                        ]
                    , optionInput Point.typeProtocol
                        "Protocol"
                        [ ( Point.valueRTU, "RTU" )
                        , ( Point.valueTCP, "TCP" )
                        ]
                    , viewIf
                        (protocol == Point.valueRTU)
                      <|
                        textInput Point.typePort "Port" "/dev/ttyUSB0"
                    , viewIf
                        (protocol
                            == Point.valueTCP
                            && clientServer
                            == Point.valueServer
                        )
                      <|
                        textInput Point.typePort "Port" "502"
                    , viewIf
                        (protocol
                            == Point.valueTCP
                            && clientServer
                            == Point.valueClient
                        )
                      <|
                        textInput Point.typeURI "URI" "192.168.1.201:502"
                    , viewIf (protocol == Point.valueRTU) <| textInput Point.typeBaud "Baud" "9600"
                    , viewIf (clientServer == Point.valueServer) <|
                        numberInput Point.typeID "Device ID"
                    , viewIf (clientServer == Point.valueClient) <|
                        numberInput Point.typePollPeriod "Poll period (ms)"
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , checkboxInput Point.typeDisabled "Disabled"
                    , counterWithReset Point.typeErrorCount Point.typeErrorCountReset "Error Count"
                    , counterWithReset Point.typeErrorCountEOF Point.typeErrorCountEOFReset "EOF Error Count"
                    , counterWithReset Point.typeErrorCountCRC Point.typeErrorCountCRCReset "CRC Error Count"
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
