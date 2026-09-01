module Components.NodeSync exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import UI.Form as Form
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
                    , viewIf (authToken == "") <| viewDeviceKey o pubKey
                    , checkboxInput Point.typeDisabled "Disabled"
                    , counterWithReset Point.typeSyncCount Point.typeSyncCountReset "Sync Count"
                    ]

                else
                    []
               )


{-| viewDeviceKey shows the key this instance connects with when no token is
set, and takes a seed issued by the upstream. The seed goes to the key file
through the API rather than becoming a point, since points replicate upstream.
-}
viewDeviceKey : NodeOptions msg -> String -> Element msg
viewDeviceKey o pubKey =
    column [ spacing 6, paddingXY 0 6 ]
        [ row [ spacing 6 ]
            [ text "Device key:"
            , el [ Font.family [ Font.monospace ], Font.size 14 ] <|
                text <|
                    if pubKey == "" then
                        "(not yet reported)"

                    else
                        pubKey
            ]
        , text "Enroll this key on the upstream, or install a key the upstream issued:"
        , row [ spacing 6 ]
            [ Input.text [ width (px 560), Font.family [ Font.monospace ], Font.size 14 ]
                { onChange = o.onEditScratch
                , text = o.scratch
                , placeholder = Just <| Input.placeholder [] <| text "seed from siot key gen"
                , label = Input.labelHidden "seed"
                }
            , Form.button
                { label = "Install key"
                , color = colors.blue
                , onPress = o.onInstallDeviceKey o.scratch
                }
            ]
        ]
