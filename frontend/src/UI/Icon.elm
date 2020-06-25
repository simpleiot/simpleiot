module UI.Icon exposing
    ( check
    , cloud
    , cloudOff
    , plus
    , power
    , powerOff
    , userCheck
    , userMinus
    , userPlus
    , userX
    , x
    )

import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Input as Input
import FeatherIcons
import Global exposing (Msg)
import Svg
import Svg.Attributes exposing (..)
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


icon : FeatherIcons.Icon -> Element msg
icon iconIn =
    el [ padding 5 ] <| html <| FeatherIcons.toHtml [] iconIn


x : msg -> Element msg
x msg =
    button FeatherIcons.x msg


check : msg -> Element msg
check msg =
    button FeatherIcons.check msg


plus : msg -> Element msg
plus msg =
    button FeatherIcons.plus msg


userPlus : msg -> Element msg
userPlus msg =
    button FeatherIcons.userPlus msg


userX : msg -> Element msg
userX msg =
    button FeatherIcons.userX msg


userMinus : msg -> Element msg
userMinus msg =
    button FeatherIcons.userMinus msg


userCheck : msg -> Element msg
userCheck msg =
    button FeatherIcons.userCheck msg


cloudOff : Element msg
cloudOff =
    icon FeatherIcons.cloudOff


cloud : Element msg
cloud =
    icon FeatherIcons.cloud


power : Element msg
power =
    icon FeatherIcons.power



-- powerOff icon is not finished yet


powerOff : Element Msg
powerOff =
    [ Svg.path [ d "M18.36 6.64a9 9 0 1 1-12.73 0" ] []
    , Svg.line [ x1 "12", y1 "2", x2 "12", y2 "12" ] []
    ]
        |> FeatherIcons.customIcon
        |> icon
