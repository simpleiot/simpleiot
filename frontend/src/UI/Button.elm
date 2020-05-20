module UI.Button exposing (view)

import Element exposing (..)
import Element.Input as Input
import UI.Styles as Styles


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
