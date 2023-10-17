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

                        stateDisplay =
                            String.replace "NmDeviceState" "" state
                    in
                    [ textDisplay "State" stateDisplay ]

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
