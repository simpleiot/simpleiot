module UI.Button exposing (view)

import Element exposing (..)
import Element.Input as Input
import UI.Style as Style


view :
    { color : Color
    , onPress : Maybe msg
    , label : Element msg
    }
    -> Element msg
view options =
    Input.button
        ((if options.onPress == Nothing then
            alpha 0.6

          else
            alpha 1
         )
            :: Style.button options.color
        )
        { onPress = options.onPress, label = options.label }
