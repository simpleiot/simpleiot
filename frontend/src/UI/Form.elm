module UI.Form exposing
    ( button
    , buttonRow
    , onEnter
    , onEnterEsc
    )

import Element exposing (..)
import Element.Font as Font
import Element.Input as Input
import Html.Events
import Json.Decode as Decode
import UI.Style as Style


onEnter : msg -> Element.Attribute msg
onEnter msg =
    Element.htmlAttribute
        (Html.Events.on "keyup"
            (Decode.field "key" Decode.string
                |> Decode.andThen
                    (\key ->
                        if key == "Enter" then
                            Decode.succeed msg

                        else
                            Decode.fail "Not the enter key"
                    )
            )
        )


onEnterEsc : msg -> msg -> Element.Attribute msg
onEnterEsc enterMsg escMsg =
    Element.htmlAttribute
        (Html.Events.on "keyup"
            (Decode.field "key" Decode.string
                |> Decode.andThen
                    (\key ->
                        if key == "Enter" then
                            Decode.succeed enterMsg

                        else if key == "Escape" then
                            Decode.succeed escMsg

                        else
                            Decode.fail "Not the enter key"
                    )
            )
        )


buttonRow : List (Element msg) -> Element msg
buttonRow =
    row
        [ Font.size 16
        , Font.bold
        , width fill
        , padding 16
        , spacing 16
        ]


button :
    { color : Color
    , onPress : msg
    , label : String
    }
    -> Element msg
button options =
    Input.button
        (Style.button options.color)
        { onPress = Just options.onPress
        , label = el [ centerX ] <| text options.label
        }
