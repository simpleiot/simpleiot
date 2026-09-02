module Components.NodeDeviceCred exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Time
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)
import Utils.Iso8601 exposing (toDateTimeString)


{-| A device credential sits under a device node on the upstream and holds
the device's public key. The upstream keeps lastConnect and connected on it.
-}
view : NodeOptions msg -> Element msg
view o =
    let
        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        connected =
            Point.getBool o.node.points Point.typeConnected ""

        pending =
            Point.getBool o.node.points Point.typePending ""
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
            , viewIf pending <| el [ Font.color colors.orange ] <| text "(pending approval)"
            , viewIf (connected && not disabled && not pending) <| text "(connected)"
            ]
            :: (if o.expDetail then
                    let
                        opts =
                            oToInputO o 100

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        lastConnect =
                            Point.getValue o.node.points Point.typeLastConnect ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typePubKey "Public key" "from siot key show on the device"
                    , checkboxInput Point.typeDisabled "Disabled"
                    , viewIf pending <| checkboxInput Point.typePending "Pending"
                    , viewIf pending <| text "Uncheck Pending to approve this device."
                    , text <| "Last connect: " ++ formatLastConnect o.zone lastConnect
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
