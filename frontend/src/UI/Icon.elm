module UI.Icon exposing
    ( arrowDown
    , arrowRight
    , check
    , cloud
    , cloudOff
    , device
    , maximize
    , minimize
    , move
    , plus
    , plusCircle
    , power
    , user
    , userCheck
    , userMinus
    , userPlus
    , userX
    , users
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


plusCircle : msg -> Element msg
plusCircle msg =
    button FeatherIcons.plusCircle msg


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


move : msg -> Element msg
move msg =
    button FeatherIcons.move msg


arrowRight : msg -> Element msg
arrowRight msg =
    button FeatherIcons.arrowRight msg


arrowDown : msg -> Element msg
arrowDown msg =
    button FeatherIcons.arrowDown msg


minimize : msg -> Element msg
minimize msg =
    button FeatherIcons.minimize2 msg


maximize : msg -> Element msg
maximize msg =
    button FeatherIcons.maximize2 msg


cloudOff : Element msg
cloudOff =
    icon FeatherIcons.cloudOff


cloud : Element msg
cloud =
    icon FeatherIcons.cloud


power : Element msg
power =
    icon FeatherIcons.power


user : Element msg
user =
    icon FeatherIcons.user


users : Element msg
users =
    icon FeatherIcons.users


device : Element msg
device =
    icon FeatherIcons.hardDrive
