module Components.Button exposing (view, view2, viewRow)

import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Utils.Styles as Styles


viewRow : List (Element msg) -> Element msg
viewRow =
    row
        [ Font.size 16
        , Font.bold
        , width fill
        , padding 16
        , spacing 16
        ]


view2 : String -> Color -> msg -> Element msg
view2 lbl color action =
    Input.button
        [ Background.color color
        , padding 16
        , width fill
        , Border.rounded 6
        , Border.width 2
        ]
        { onPress = Just action
        , label = el [ centerX ] <| text lbl
        }


view :
    { onPress : Maybe msg
    , label : Element msg
    }
    -> Element msg
view config =
    Input.button
        ((if config.onPress == Nothing then
            alpha 0.6

          else
            alpha 1
         )
            :: Styles.button
        )
        config
