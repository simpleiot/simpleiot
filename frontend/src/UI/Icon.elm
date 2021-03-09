module UI.Icon exposing
    ( blank
    , bus
    , check
    , cloud
    , cloudOff
    , device
    , io
    , list
    , minus
    , power
    , send
    , trendingUp
    , user
    , users
    )

import Element exposing (..)
import FeatherIcons
import Svg
import Svg.Attributes as S


icon : FeatherIcons.Icon -> Element msg
icon iconIn =
    el [ padding 5 ] <| html <| FeatherIcons.toHtml [] iconIn



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


list : Element msg
list =
    icon FeatherIcons.list


check : Element msg
check =
    icon FeatherIcons.check


trendingUp : Element msg
trendingUp =
    icon FeatherIcons.trendingUp


send : Element msg
send =
    icon FeatherIcons.send
