module UI.Icon exposing
    ( arrowDown
    , arrowRight
    , blank
    , bus
    , check
    , cloud
    , cloudOff
    , device
    , io
    , maximize
    , message
    , minimize
    , minus
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


icon : FeatherIcons.Icon -> Element msg
icon iconIn =
    el [ padding 5 ] <| html <| FeatherIcons.toHtml [] iconIn



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


message : msg -> Element msg
message msg =
    button FeatherIcons.messageSquare msg



-- non-clickable icons


bus : Element msg
bus =
    [ Svg.line [ S.x1 "11", S.y1 "3", S.x2 "11", S.y2 "14" ] []
    , Svg.polyline [ S.points "3 14 3 9 19 9 19 14" ] []
    , Svg.rect [ S.fill "rgb(0,0,0)", S.stroke "none", S.x "0", S.y "14", S.width "6", S.height "5" ] []
    , Svg.rect [ S.fill "rgb(0,0,0)", S.stroke "none", S.x "8", S.y "14", S.width "6", S.height "5" ] []
    , Svg.rect [ S.fill "rgb(0,0,0)", S.stroke "none", S.x "16", S.y "14", S.width "6", S.height "5" ] []
    ]
        |> FeatherIcons.customIcon
        |> icon


io : Element msg
io =
    [ Svg.polyline [ S.points "3 6 3 16" ] []
    , Svg.polyline [ S.points "12 3 8 19" ] []
    , Svg.ellipse [ S.cx "18", S.cy "11", S.rx "3", S.ry "5" ] []
    ]
        |> FeatherIcons.customIcon
        |> icon


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


minus : Element msg
minus =
    icon FeatherIcons.minus


blank : Element msg
blank =
    el [ width (px 33), height (px 33) ] <| text ""
