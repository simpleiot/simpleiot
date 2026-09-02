module Components.NodeEnrollToken exposing (view)

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


{-| An enrollment token lets devices that hold it ask this instance for a
credential. Only a hash is stored; the token is shown once when generated.
-}
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
            [ Icon.key
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , text "(enrollment token)"
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    let
                        opts =
                            oToInputO o 100

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        hash =
                            Point.getText o.node.points Point.typeTokenHash ""

                        expires =
                            Point.getValue o.node.points Point.typeExpires ""

                        generated =
                            case o.generatedToken of
                                Just k ->
                                    if k.id == o.node.id then
                                        Just k.token

                                    else
                                        Nothing

                                Nothing ->
                                    Nothing
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , checkboxInput Point.typeDisabled "Disabled"
                    , checkboxInput Point.typeAutoApprove "Auto approve"
                    , text "Auto approve lets enrolled devices in without an operator."
                    , text <| "Expires: " ++ formatExpires o.zone expires
                    , text <|
                        if hash == "" then
                            "No token generated yet"

                        else
                            "Token set (only its hash is stored)"
                    , viewIf (hash == "") <|
                        Form.buttonRow
                            [ Form.button
                                { label = "Generate token"
                                , color = colors.blue
                                , onPress = o.onGenerateKey
                                }
                            ]
                    , case generated of
                        Just token ->
                            viewToken token

                        Nothing ->
                            none
                    ]

                else
                    []
               )


formatExpires : Time.Zone -> Float -> String
formatExpires zone t =
    if t <= 0 then
        "never"

    else
        toDateTimeString zone (Time.millisToPosix (round (t * 1000)))


viewToken : String -> Element msg
viewToken token =
    column
        [ spacing 6
        , padding 10
        , Border.width 1
        , Border.color colors.orange
        ]
        [ text "Enrollment token. It is shown once and not stored here."
        , el [ Font.family [ Font.monospace ], Font.size 14 ] <| text token
        , text "Put it on each device's sync node as the Enroll Token."
        ]
