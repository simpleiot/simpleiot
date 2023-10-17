module Components.NodeNetworkManagerConn exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style


view : NodeOptions msg -> Element msg
view o =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.cable
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        textInputKey =
                            NodeInputs.nodeTextInput opts

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        checkboxInputKey =
                            NodeInputs.nodeCheckboxInput opts

                        numberInput =
                            NodeInputs.nodeNumberInput opts "0"

                        numberInputKey =
                            NodeInputs.nodeNumberInput opts
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , checkboxInput "disabled" "Disabled"
                    , textInput "interface" "Interface" ""
                    , checkboxInput "autoConnect" "Auto connect"
                    , numberInput "autoConnectPriority" "Auto conn priority"
                    , el [ Font.bold, centerX ] <| text "IPv4 Settings"
                    , checkboxInputKey "ipv4Config" "staticIP" "Static IP"
                    , textInputKey "ipv4Config" "address" "Address" "1.2.3.4"
                    , textInputKey "ipv4Config" "netmask" "Netmask" "255.255.0.0"
                    , textInputKey "ipv4Config" "gateway" "Gateway" "10.0.0.1"
                    , textInputKey "ipv4Config" "dnsServer1" "DNS Server 1" "8.8.8.8"
                    , textInputKey "ipv4Config" "dnsServer2" "DNS Server 2" "8.8.4.4"
                    , el [ Font.bold, centerX ] <| text "IPv6 Settings"
                    , checkboxInputKey "ipv6Config" "staticIP" "Static IP"
                    , textInputKey "ipv6Config" "address" "Address" ""
                    , numberInputKey "ipv6Config" "prefix" "Prefix"
                    , textInputKey "ipv6Config" "gateway" "Gateway" ""
                    , textInputKey "ipv6Config" "dnsServer1" "DNS Server 1" ""
                    , textInputKey "ipv6Config" "dnsServer2" "DNS Server 2" ""
                    ]

                else
                    []
               )
