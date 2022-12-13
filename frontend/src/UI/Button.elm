module UI.Button exposing
    ( arrowDown
    , arrowRight
    , clipboard
    , copy
    , dot
    , message
    , plusCircle
    , x
    )

import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Input as Input
import FeatherIcons
import Svg
import Svg.Attributes as S
import UI.Style as Style


button : FeatherIcons.Icon -> msg -> Element msg
button iconIn msg =
    Input.button
        [ padding 5
        , Border.rounded 50
        , mouseOver
            [ Background.color Style.colors.ltgray
            ]

        --, Element.focused
        --    [ Background.color Style.palette.ltgray
        --    ]
        ]
        { onPress = Just msg
        , label =
            html
                (FeatherIcons.toHtml [] iconIn)
        }



-- Button Icons


x : msg -> Element msg
x msg =
    button FeatherIcons.x msg


plusCircle : msg -> Element msg
plusCircle msg =
    button FeatherIcons.plusCircle msg


arrowRight : msg -> Element msg
arrowRight msg =
    button FeatherIcons.chevronRight msg


arrowDown : msg -> Element msg
arrowDown msg =
    button FeatherIcons.chevronDown msg


message : msg -> Element msg
message msg =
    button FeatherIcons.messageSquare msg


copy : msg -> Element msg
copy msg =
    button FeatherIcons.copy msg


clipboard : msg -> Element msg
clipboard msg =
    button FeatherIcons.clipboard msg


dot : msg -> Element msg
dot =
    [ Svg.circle
        [ S.style "fill:#000000;fill-opacity:1;"
        , S.cx "11.903377"
        , S.cy "11.823219"
        , S.r "4.1"
        ]
        []
    ]
        |> FeatherIcons.customIcon
        |> button
