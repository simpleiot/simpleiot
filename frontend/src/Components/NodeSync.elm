module Components.NodeSync exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        error =
            Point.getText o.node.points Point.typeError ""
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
            , viewIf (error /= "") <| el [ Font.color colors.red ] <| text <| "(" ++ error ++ ")"
            ]
            :: (if o.expDetail then
                    let
                        opts =
                            oToInputO o 100

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts "0"

                        authToken =
                            Point.getText o.node.points Point.typeAuthToken ""

                        pubKey =
                            Point.getText o.node.points Point.typePubKey ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeURI "URI" "nats://myserver:4222, ws://myserver"
                    , textInput Point.typeAuthToken "Auth Token" ""
                    , viewIf (authToken == "") <| viewDeviceKey pubKey
                    , viewIf (authToken == "") <| textInput Point.typeEnrollToken "Enroll Token" "optional, from the upstream"
                    , checkboxInput Point.typeDisabled "Disabled"
                    , counterWithReset Point.typeSyncCount Point.typeSyncCountReset "Sync Count"
                    ]

                else
                    []
               )


{-| viewDeviceKey shows the key this instance connects with when no token is
set, which is what an upstream enrolls.
-}
viewDeviceKey : String -> Element msg
viewDeviceKey pubKey =
    row [ spacing 6, paddingXY 0 6 ]
        [ text "Device key:"
        , el [ Font.family [ Font.monospace ], Font.size 14 ] <|
            text <|
                if pubKey == "" then
                    "(not yet reported)"

                else
                    pubKey
        ]
