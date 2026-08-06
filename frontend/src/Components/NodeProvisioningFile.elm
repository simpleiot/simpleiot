module Components.NodeProvisioningFile exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import UI.Icon as Icon
import UI.Style exposing (colors)
import Utils.Iso8601 exposing (toDateTimeString)


{-| A provisioning file on disk, and what provisioning last did with it. This
is a status rather than a setting: everything here is recorded by the server,
so nothing on it is editable.
-}
view : NodeOptions msg -> Element msg
view o =
    let
        name =
            Point.getText o.node.points Point.typeDescription ""

        err =
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
            [ Icon.file
            , text name
            , viewError err
            ]
            :: (if o.expDetail then
                    [ viewStatus o ]

                else
                    []
               )


{-| viewError says at a glance whether a file applied, without expanding it.
-}
viewError : String -> Element msg
viewError err =
    if err == "" then
        none

    else
        el [ Font.color colors.red ] <| text err


{-| viewStatus shows what was applied and when. The hash is what provisioning
compares to decide whether there is anything to do, and the timestamp on that
point is when it last did it.
-}
viewStatus : NodeOptions msg -> Element msg
viewStatus o =
    let
        hash =
            Point.getText o.node.points Point.typeHash ""

        applied =
            case Point.get o.node.points Point.typeHash "" of
                Just p ->
                    toDateTimeString o.zone p.time

                Nothing ->
                    "never"
    in
    column [ spacing 6, paddingEach { top = 0, bottom = 0, left = 70, right = 0 } ]
        [ text <| "Applied: " ++ applied
        , text <| "Contents: " ++ shortHash hash
        ]


{-| shortHash keeps a checksum readable. The whole thing says nothing more to a
person than its first few characters do.
-}
shortHash : String -> String
shortHash hash =
    if String.length hash > 12 then
        String.left 12 hash

    else if hash == "" then
        "unknown"

    else
        hash
