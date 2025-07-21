module UI.Icon exposing
    ( activity
    , barChart
    , blank
    , bus
    , cable
    , check
    , clipboard
    , clock
    , cloud
    , cloudOff
    , database
    , device
    , file
    , globe
    , io
    , list
    , network
    , oneWire
    , particle
    , power
    , radioReceiver
    , send
    , serialDev
    , shelly
    , sync
    , trendingDown
    , trendingUp
    , update
    , user
    , users
    , variable
    )

import Element exposing (..)
import FeatherIcons
import Svg
import Svg.Attributes as S


icon : FeatherIcons.Icon -> Element msg
icon iconIn =
    el [ padding 5 ] <| html <| FeatherIcons.toHtml [] iconIn



-- non-clickable FeatherIcons


radioReceiver : Element msg
radioReceiver =
    [ Svg.path [ S.d "M5 16v2" ] []
    , Svg.path [ S.d "M19 16v2" ] []
    , Svg.rect [ S.width "20", S.height "8", S.x "2", S.y "8", S.rx "2" ] []
    , Svg.path [ S.d "M18 12h0" ] []
    ]
        |> FeatherIcons.customIcon
        |> icon


network : Element msg
network =
    [ Svg.rect [ S.x "16", S.y "16", S.width "6", S.height "6", S.rx "1" ] []
    , Svg.rect [ S.x "2", S.y "16", S.width "6", S.height "6", S.rx "1" ] []
    , Svg.rect [ S.x "9", S.y "2", S.width "6", S.height "6", S.rx "1" ] []
    , Svg.path [ S.d "M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3" ] []
    , Svg.path [ S.d "M12 12V8" ] []
    ]
        |> FeatherIcons.customIcon
        |> icon


cable : Element msg
cable =
    [ Svg.path [ S.d "M4 9a2 2 0 0 1-2-2V5h6v2a2 2 0 0 1-2 2Z" ] []
    , Svg.path [ S.d "M3 5V3" ] []
    , Svg.path [ S.d "M7 5V3" ] []
    , Svg.path [ S.d "M19 15V6.5a3.5 3.5 0 0 0-7 0v11a3.5 3.5 0 0 1-7 0V9" ] []
    , Svg.path [ S.d "M17 21v-2" ] []
    , Svg.path [ S.d "M21 21v-2" ] []
    , Svg.path [ S.d "M22 19h-6v-2a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2" ] []
    ]
        |> FeatherIcons.customIcon
        |> icon


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


particle : Element msg
particle =
    [ Svg.g [ S.style "stroke-width:0;fill:#000000" ]
        [ Svg.polyline [ S.points "3 11 11 11 11 7  13 7  13 11 21 11 12 2  3 11" ] []
        , Svg.polyline [ S.points "3 13 11 13 11 17 13 17 13 13 21 13 12 22 3 13" ] []
        ]
    ]
        |> FeatherIcons.customIcon
        |> icon


shelly : Element msg
shelly =
    [ Svg.g [ S.style "stroke-width:0;fill:#000000" ]
        [ Svg.path [ S.d "M12 0C5.373 0 0 5.373 0 12a12 12 0 0 0 .033.88c1.07-.443 2.495-.679 4.322-.679h5.762c-.167.61-.548 1.087-1.142 1.436-.532.308-1.14.463-1.823.463h-.927c-.89 0-1.663.154-2.32.463-.859.403-1.286 1-1.286 1.789 0 .893.59 1.594 1.774 2.1a7.423 7.423 0 0 0 2.927.581c1.318 0 2.416-.29 3.297-.867 1.024-.664 1.535-1.616 1.535-2.857 0-.854-.325-2.08-.976-3.676-.65-1.597-.975-2.837-.975-3.723 0-2.79 2.305-4.233 6.916-4.324.641-.01 1.337-.005 1.916-.004.593 0 1.144.05 1.66.147A12 12 0 0 0 12 0zm4.758 5.691c-1.206 0-1.809.502-1.809 1.506 0 .514.356 1.665 1.067 3.451.71 1.787 1.064 3.186 1.064 4.198 0 2.166-1.202 3.791-3.607 4.875-1.794.797-3.892 1.197-6.297 1.197-1.268 0-2.442-.114-3.543-.316A12 12 0 0 0 12 24c6.627 0 12-5.373 12-12a12 12 0 0 0-.781-4.256 3.404 3.404 0 0 1-.832.77h-4.371l1.425-2.828a299.94 299.94 0 0 0-2.683.005Z" ] []
        ]
    ]
        |> FeatherIcons.customIcon
        |> icon


variable : Element msg
variable =
    [ Svg.g [ S.transform "scale(3.3,4.7)", S.style "stroke-width:0.25;fill:#000000" ]
        [ Svg.path [ S.d "m 6.0407008,4.5926947 q -0.048802,0.028837 -0.095385,0.00222 -0.044365,-0.026619 -0.075421,-0.077639 -0.031056,-0.05102 -0.03771,-0.1064766 -0.00665,-0.053238 0.028837,-0.077639 Q 6.025173,4.2244632 6.1871061,4.0714031 6.3490392,3.9183431 6.4754802,3.7164812 6.6041393,3.5146194 6.6839968,3.2617376 q 0.079857,-0.2551001 0.079857,-0.5634385 0,-0.3083384 -0.079857,-0.5634385 Q 6.6041393,1.8797604 6.4754802,1.6756804 6.3490392,1.4716003 6.1871061,1.3185402 6.025173,1.1632619 5.8610216,1.054567 q -0.035492,-0.024401 -0.028837,-0.0776391 0.00665,-0.0554565 0.03771,-0.10647656 0.031056,-0.05102 0.075421,-0.0776392 0.046584,-0.0266191 0.095385,0.002218 0.186334,0.1109131 0.372668,0.29059229 0.1885523,0.1796792 0.3393941,0.4214698 0.1508418,0.2395722 0.2440088,0.5390376 0.095385,0.2994653 0.095385,0.652169 0,0.3527036 -0.095385,0.6499507 Q 6.9036047,3.6454969 6.7527629,3.8850691 6.6019211,4.1246414 6.4133688,4.3021024 6.2270348,4.4817816 6.0407008,4.5926947 Z" ] []
        , Svg.path [ S.d "M 3.4652989,2.3389406 2.7732012,1.3806515 H 2.6201411 q -0.075421,0 -0.1086948,-0.033274 -0.031056,-0.033274 -0.031056,-0.1175679 0,-0.084294 0.031056,-0.1175678 0.033274,-0.033274 0.1086948,-0.033274 h 0.7630821 q 0.075421,0 0.1064766,0.033274 0.033274,0.033274 0.033274,0.1175678 0,0.084294 -0.033274,0.1175679 -0.031056,0.033274 -0.1064766,0.033274 H 3.1857979 L 3.6760338,2.0816223 4.1684879,1.3806515 H 4.0353922 q -0.075421,0 -0.1086948,-0.033274 -0.031056,-0.033274 -0.031056,-0.1175679 0,-0.084294 0.031056,-0.1175678 0.033274,-0.033274 0.1086948,-0.033274 h 0.6743516 q 0.075421,0 0.1064765,0.033274 0.033274,0.033274 0.033274,0.1175678 0,0.084294 -0.033274,0.1175679 -0.031056,0.033274 -0.1064765,0.033274 H 4.5655568 L 3.8801139,2.3433772 4.6476325,3.4103611 h 0.1286591 q 0.075421,0 0.1064766,0.033274 0.033274,0.033274 0.033274,0.1175679 0,0.084294 -0.033274,0.1175679 -0.031056,0.033274 -0.1064766,0.033274 H 3.9910269 q -0.075421,0 -0.1086948,-0.033274 -0.031056,-0.033274 -0.031056,-0.1175679 0,-0.084294 0.031056,-0.1175679 0.033274,-0.033274 0.1086948,-0.033274 H 4.2328175 L 3.6671607,2.6006955 3.0881944,3.4103611 h 0.1841157 q 0.075421,0 0.1064766,0.033274 0.033274,0.033274 0.033274,0.1175679 0,0.084294 -0.033274,0.1175679 -0.031056,0.033274 -0.1064766,0.033274 H 2.5535933 q -0.075421,0 -0.1086948,-0.033274 -0.031056,-0.033274 -0.031056,-0.1175679 0,-0.084294 0.031056,-0.1175679 0.033274,-0.033274 0.1086948,-0.033274 h 0.1375322 z" ] []
        , Svg.path [ S.d "M 1.2891841,4.5926947 Q 1.1028501,4.4817816 0.91429783,4.3021024 0.72796384,4.1246414 0.57712203,3.8850691 0.42628023,3.6454969 0.33089497,3.3482498 q -0.093167,-0.2972471 -0.093167,-0.6499507 0,-0.3527037 0.093167,-0.652169 Q 0.42628023,1.7466647 0.57712203,1.5070925 0.72796384,1.2653019 0.91429783,1.0856227 1.1028501,0.90594351 1.2891841,0.79503041 q 0.048802,-0.0288374 0.093167,-0.002218 0.046584,0.0266191 0.077639,0.0776392 0.031056,0.05102 0.03771,0.10647656 0.00665,0.0532383 -0.028837,0.0776391 -0.1641514,0.1086949 -0.3260845,0.2639732 -0.16193311,0.1530601 -0.2905923,0.3571402 -0.12865919,0.20408 -0.20851661,0.4591802 -0.0776392,0.2551001 -0.0776392,0.5634385 0,0.3083384 0.0776392,0.5634385 0.0798574,0.2528818 0.20851661,0.4547436 0.12865919,0.2018619 0.2905923,0.3549219 0.1619331,0.1530601 0.3260845,0.2617549 0.035492,0.024401 0.028837,0.077639 -0.00665,0.055457 -0.03771,0.1064766 -0.031056,0.05102 -0.077639,0.077639 -0.044365,0.026619 -0.093167,-0.00222 z" ] []
        ]
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


oneWire : Element msg
oneWire =
    [ Svg.g [ S.transform "scale(0.6,0.8)", S.style "stroke-width:1;fill:#000000" ]
        [ Svg.path [ S.d "m 13.528858,21.000319 q 0.498704,0 0.704053,0.249352 0.220016,0.234684 0.220016,0.806728 0,0.572043 -0.220016,0.821395 -0.205349,0.234684 -0.704053,0.234684 H 3.6427783 q -0.4987043,0 -0.7187209,-0.234684 -0.2053489,-0.249352 -0.2053489,-0.821395 0,-0.572044 0.2053489,-0.806728 0.2200166,-0.249352 0.7187209,-0.249352 H 7.5884094 V 8.5327113 L 3.9361337,10.806216 q -0.2493521,0.146678 -0.4547009,0.146678 -0.4693688,0 -0.8067276,-0.601379 -0.2053488,-0.3520264 -0.2053488,-0.6747174 0,-0.4987043 0.454701,-0.7627242 L 8.453808,5.5111499 q 0.2933555,-0.1760133 0.586711,-0.1760133 0.322691,0 0.5280398,0.2200166 0.2200166,0.2053489 0.2200166,0.586711 V 21.000319 Z" ] []
        , Svg.path [ S.d "m 33.169006,5.569821 q 0.52804,0.073339 0.762725,0.2640199 0.234684,0.1906811 0.234684,0.5573754 0,0.1760133 -0.01467,0.26402 L 31.966249,22.481764 q -0.07334,0.52804 -0.381362,0.733389 -0.293356,0.220016 -0.880067,0.220016 -1.129418,0 -1.466777,-0.953405 l -3.036229,-8.595315 -3.036229,8.595315 q -0.337359,0.953405 -1.466778,0.953405 -0.586711,0 -0.894734,-0.220016 -0.293355,-0.205349 -0.366694,-0.733389 L 18.251881,6.6552363 q -0.01467,-0.088007 -0.01467,-0.26402 0,-0.3666943 0.234684,-0.5573754 0.234685,-0.190681 0.762724,-0.2640199 0.102675,-0.014668 0.293356,-0.014668 0.440033,0 0.645382,0.2053489 0.205349,0.190681 0.26402,0.645382 L 22.08017,19.328193 25.116399,10.71821 q 0.102674,-0.293356 0.352026,-0.440034 0.26402,-0.146677 0.733389,-0.146677 0.469369,0 0.718721,0.146677 0.26402,0.146678 0.366694,0.440034 l 3.036229,8.609983 1.642791,-12.9223089 q 0.05867,-0.454701 0.26402,-0.645382 0.205349,-0.2053489 0.645382,-0.2053489 0.190681,0 0.293355,0.014668 z" ] []
        ]
    ]
        |> FeatherIcons.customIcon
        |> icon


serialDev : Element msg
serialDev =
    [ Svg.g [ S.transform "scale(1.0,1.5),translate(0,-4)", S.style "stroke-width:1.5" ]
        [ Svg.path [ S.d "m 1,9 h 22.428778 l -3.364317,5.685801 H 4.3643165 Z" ] []
        , Svg.ellipse [ S.style "stroke-width:0.8", S.cx "5.48", S.cy "10.75", S.rx "0.56", S.ry "0.57" ] []
        , Svg.ellipse [ S.style "stroke-width:0.8", S.cx "8.85", S.cy "10.75", S.rx "0.56", S.ry "0.57" ] []
        , Svg.ellipse [ S.style "stroke-width:0.8", S.cx "12.21", S.cy "10.75", S.rx "0.56", S.ry "0.57" ] []
        , Svg.ellipse [ S.style "stroke-width:0.8", S.cx "15.58", S.cy "10.75", S.rx "0.56", S.ry "0.57" ] []
        , Svg.ellipse [ S.style "stroke-width:0.8", S.cx "18.94", S.cy "10.75", S.rx "0.56", S.ry "0.57" ] []
        , Svg.ellipse [ S.style "stroke-width:0.8", S.cx "7.16", S.cy "12.41", S.rx "0.56", S.ry "0.57" ] []
        , Svg.ellipse [ S.style "stroke-width:0.8", S.cx "10.53", S.cy "12.41", S.rx "0.56", S.ry "0.57" ] []
        , Svg.ellipse [ S.style "stroke-width:0.8", S.cx "13.90", S.cy "12.41", S.rx "0.56", S.ry "0.57" ] []
        , Svg.ellipse [ S.style "stroke-width:0.8", S.cx "17.26", S.cy "12.41", S.rx "0.56", S.ry "0.57" ] []
        ]
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


activity : Element msg
activity =
    icon FeatherIcons.activity


trendingDown : Element msg
trendingDown =
    icon FeatherIcons.trendingDown


send : Element msg
send =
    icon FeatherIcons.send


sync : Element msg
sync =
    icon FeatherIcons.refreshCw


database : Element msg
database =
    icon FeatherIcons.database


clipboard : Element msg
clipboard =
    icon FeatherIcons.clipboard


barChart : Element msg
barChart =
    icon FeatherIcons.barChart2


file : Element msg
file =
    icon FeatherIcons.file


clock : Element msg
clock =
    icon FeatherIcons.clock


update : Element msg
update =
    icon FeatherIcons.refreshCw

globe : Element msg
globe =
    icon FeatherIcons.globe
