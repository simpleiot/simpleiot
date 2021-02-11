module UI.Style exposing
    ( button
    , colors
    , error
    , fonts
    , h1
    , h1w
    , h2
    , h3
    , h4
    , h5
    , h6
    , link
    , size
    , transition
    )

import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Html.Attributes as Attr


colors :
    { white : Color
    , jet : Color
    , coral : Color
    , black : Color
    , ltgray : Color
    , gray : Color
    , darkgray : Color
    , pale : Color
    , red : Color
    , orange : Color
    , yellow : Color
    , green : Color
    , blue : Color
    , none : Color
    }
colors =
    { white = rgb 1 1 1
    , jet = rgb255 40 40 40
    , coral = rgb255 204 75 75
    , black = rgb 0 0 0
    , ltgray = rgb 0.9 0.9 0.9
    , gray = rgb 0.5 0.5 0.5
    , darkgray = rgb 0.8 0.8 0.8
    , pale = rgba 0.97 0.97 0.97 0.9
    , red = rgb255 204 85 68
    , orange = rgb 1 1 0.8
    , yellow = rgb 1 1 0.7
    , green = rgba 0.7 1 0.7 0.9
    , blue = rgb255 50 100 150
    , none = rgba 0 0 0 0
    }


fonts : { sans : List Font.Font }
fonts =
    { sans =
        [ Font.external
            { name = "IBM Plex Sans"
            , url = "https://fonts.googleapis.com/css?family=IBM+Plex+Sans:400,400i,600,600i&display=swap"
            }
        , Font.serif
        ]
    }


size : { body : Attr dec msg, heading : Attr dec msg }
size =
    { body = Font.size 16
    , heading = Font.size 20
    }


link : List (Attribute msg)
link =
    [ Font.underline
    , Font.color colors.blue
    , transition
        { property = "opacity"
        , duration = 150
        }
    , mouseOver
        [ alpha 0.6
        ]
    ]


button : Color -> List (Attribute msg)
button color =
    [ paddingXY 16 8
    , Font.size 14
    , Border.color color
    , Font.color color
    , Background.color colors.white
    , Border.width 2
    , Border.rounded 4
    , pointer
    , transition
        { property = "all"
        , duration = 150
        }
    , mouseOver
        [ Font.color colors.white
        , Background.color color
        ]
    ]


error : List (Attribute msg)
error =
    [ paddingXY 16 8
    , Font.size 14
    , Font.color colors.white
    , Font.bold
    , Background.color colors.coral
    , Border.width 2
    , Border.rounded 4
    , width fill
    ]


h1 : List (Attribute msg)
h1 =
    [ Font.family fonts.sans
    , Font.semiBold
    , Font.size 32
    ]


h2 : List (Attribute msg)
h2 =
    [ Font.family fonts.sans
    , Font.semiBold
    , Font.size 24
    ]


h3 : List (Attribute msg)
h3 =
    [ Font.family fonts.sans
    , Font.semiBold
    , Font.size 19
    ]


h4 : List (Attribute msg)
h4 =
    [ Font.family fonts.sans
    , Font.semiBold
    , Font.size 16
    ]


h5 : List (Attribute msg)
h5 =
    [ Font.family fonts.sans
    , Font.semiBold
    , Font.size 13
    ]


h6 : List (Attribute msg)
h6 =
    [ Font.family fonts.sans
    , Font.semiBold
    , Font.size 11
    ]


h1w : List (Attribute msg) -> Element msg -> Element msg
h1w =
    elWith h1


transition :
    { property : String
    , duration : Int
    }
    -> Attribute msg
transition { property, duration } =
    Element.htmlAttribute
        (Attr.style
            "transition"
            (property ++ " " ++ String.fromInt duration ++ "ms ease-in-out")
        )


elWith : List (Attribute msg) -> List (Attribute msg) -> Element msg -> Element msg
elWith styles otherStyles =
    el (Element.htmlAttribute (Attr.class "markdown") :: styles ++ otherStyles)
