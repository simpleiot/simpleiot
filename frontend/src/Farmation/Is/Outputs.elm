module Farmation.Is.Outputs exposing (Msg(..), relay, statusLed)

import Html
import Svg exposing (..)
import Svg.Attributes exposing (..)


type Msg
    = NoOp


statusLed : Bool -> Bool -> Html.Html Msg
statusLed red green =
    let
        color =
            if red then
                "red"

            else if green then
                "green"

            else
                "grey"
    in
    svg
        [ width "55"
        , height "55"
        ]
        [ circle
            [ cx "27"
            , cy "27"
            , r "25"
            , fill color
            , stroke "black"
            , strokeWidth "2"
            ]
            []
        ]


relay : String -> Bool -> Html.Html Msg
relay desc on =
    let
        color =
            if on then
                "#147ffb"

            else
                "#6c757d"
    in
    svg
        [ width "60"
        , height "30"
        ]
        [ rect
            [ x "3"
            , y "3"
            , width "54"
            , height "24"
            , fill color
            , stroke "black"
            , strokeWidth "2"
            ]
            []
        , text_
            [ x "7"
            , y "20"
            , fill "white"
            ]
            [ text desc ]
        ]
