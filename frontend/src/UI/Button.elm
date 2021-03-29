module UI.Button exposing
    ( arrowDown
    , arrowRight
    , check
    , close
    , copy
    , edit
    , maximize
    , message
    , minimize
    , move
    , plus
    , plusCircle
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



-- Button Icons


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
    button FeatherIcons.chevronRight msg


arrowDown : msg -> Element msg
arrowDown msg =
    button FeatherIcons.chevronDown msg


minimize : msg -> Element msg
minimize msg =
    button FeatherIcons.minimize2 msg


maximize : msg -> Element msg
maximize msg =
    button FeatherIcons.maximize2 msg


edit : msg -> Element msg
edit msg =
    button FeatherIcons.edit3 msg


close : msg -> Element msg
close msg =
    button FeatherIcons.minimize msg


message : msg -> Element msg
message msg =
    button FeatherIcons.messageSquare msg


copy : msg -> Element msg
copy msg =
    button FeatherIcons.copy msg
