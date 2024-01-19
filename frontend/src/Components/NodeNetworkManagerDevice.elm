module Components.NodeNetworkManagerDevice exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
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
        let
            interface =
                Point.getText o.node.points "interface" ""
        in
        wrappedRow [ spacing 10 ]
            [ Icon.radioReceiver
            , text <|
                interface
            ]
            :: (if o.expDetail then
                    let
                        state =
                            Point.getText o.node.points "state" ""

                        ipv4Gateway =
                            Point.getText o.node.points "ipv4Gateway" ""

                        stateDisplay =
                            String.replace "NmDeviceState" "" state

                        ipv4Addresses =
                            Point.getTextArray o.node.points "ipv4Addresses"

                        ipv4Netmasks =
                            Point.getTextArray o.node.points "ipv4Netmasks"

                        ipv4Nameservers =
                            Point.getTextArray o.node.points "ipv4Nameservers"
                    in
                    [ textDisplay "State" stateDisplay
                    , textDisplay "IPv4 Gateway" ipv4Gateway
                    ]
                        ++ List.map
                            (\a ->
                                textDisplay "IPv4 Addr" a
                            )
                            ipv4Addresses
                        ++ List.map
                            (\a ->
                                textDisplay "IPv4 Netmask" a
                            )
                            ipv4Netmasks
                        ++ List.map
                            (\a ->
                                textDisplay "IPv4 NameServers" a
                            )
                            ipv4Nameservers

                else
                    []
               )


textDisplay : String -> String -> Element msg
textDisplay label value =
    el [ paddingEach { top = 0, right = 0, bottom = 0, left = 50 } ] <|
        text <|
            label
                ++ ": "
                ++ value
