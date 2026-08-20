module Components.NodeMqtt exposing (view)

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
            [ Icon.mqtt
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

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        error =
                            Point.getText o.node.points Point.typeError ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeURI "Broker URI" "blank uses the built-in broker"
                    , checkboxInput Point.typeSparkplug "Sparkplug B"
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , checkboxInput Point.typeDisabled "Disabled"
                    , viewIf (error /= "") <|
                        el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
                            text <|
                                "Error: "
                                    ++ error
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
