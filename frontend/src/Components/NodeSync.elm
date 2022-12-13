module Components.NodeSync exposing (view)

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
            [ Icon.sync
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    let
                        opts =
                            oToInputO o 100

                        textInput =
                            NodeInputs.nodeTextInput opts ""

                        textNumber =
                            NodeInputs.nodeNumberInput opts ""

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts ""

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeURI "URI" "nats://myserver:4222, ws://myserver"
                    , textInput Point.typeAuthToken "Auth Token" ""
                    , textNumber Point.typePeriod "Sync Period (s)"
                    , checkboxInput Point.typeDisable "Disable"
                    , counterWithReset Point.typeSyncCount Point.typeSyncCountReset "Sync Count"
                    ]

                else
                    []
               )
