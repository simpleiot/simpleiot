module Components.NodeSparkplugNode exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)


view : NodeOptions msg -> Element msg
view o =
    let
        sysStateIcon =
            case Point.getText o.node.points Point.typeSysState "" of
                "offline" ->
                    Icon.cloudOff

                "online" ->
                    Icon.cloud

                _ ->
                    Element.none
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.radioReceiver
            , sysStateIcon
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

                        identity =
                            Point.getText o.node.points Point.typeID ""
                    in
                    [ el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
                        text <|
                            "Sparkplug edge node: "
                                ++ identity
                    , textInput Point.typeDescription "Description" ""
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
