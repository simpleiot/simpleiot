module UI.Icon exposing
    ( check
    , cloud
    , cloudOff
    , plus
    , power
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
