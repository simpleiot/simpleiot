module Components.NodeDeviceCred exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)
import Utils.Iso8601 exposing (toDateTimeString)


{-| A device credential sits under a device node on the upstream and holds
the device's public key. The upstream keeps lastConnect and connected on it,
and a key can be generated here for a device that does not have one yet, in
which case the seed is shown once.
-}
view : NodeOptions msg -> Element msg
view o =
    let
        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        connected =
            Point.getBool o.node.points Point.typeConnected ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.key
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf disabled <| text "(disabled)"
            , viewIf (connected && not disabled) <| text "(connected)"
            ]
            :: (if o.expDetail then
                    let
                        opts =
                            oToInputO o 100

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        pubKey =
                            Point.getText o.node.points Point.typePubKey ""

                        lastConnect =
                            Point.getValue o.node.points Point.typeLastConnect ""

                        generated =
                            case o.generatedKey of
                                Just k ->
                                    if k.id == o.node.id then
                                        Just k

                                    else
                                        Nothing

                                Nothing ->
                                    Nothing
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typePubKey "Public key" "from siot key show on the device"
                    , checkboxInput Point.typeDisabled "Disabled"
                    , text <| "Last connect: " ++ formatLastConnect o.zone lastConnect
                    , viewIf (pubKey == "") <|
                        Form.buttonRow
                            [ Form.button
                                { label = "Generate key"
                                , color = colors.blue
                                , onPress = o.onGenerateKey
                                }
                            ]
                    , case generated of
                        Just k ->
                            viewSeed k.seed

                        Nothing ->
                            none
                    ]

                else
                    []
               )


formatLastConnect : Time.Zone -> Float -> String
formatLastConnect zone t =
    if t <= 0 then
        "never"

    else
        toDateTimeString zone (Time.millisToPosix (round (t * 1000)))


{-| viewSeed shows a generated seed once. Nothing stores it, so it is gone
when the page is left.
-}
viewSeed : String -> Element msg
viewSeed seed =
    column
        [ spacing 6
        , padding 10
        , Border.width 1
        , Border.color colors.orange
        ]
        [ text "Seed for the device. It is shown once and not stored here."
        , el [ Font.family [ Font.monospace ], Font.size 14 ] <| text seed
        , text <| "Install it on the device with: siot key install " ++ seed
        ]
