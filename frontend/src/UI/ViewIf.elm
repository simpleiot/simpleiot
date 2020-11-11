module UI.ViewIf exposing (viewIf)

import Element exposing (..)


viewIf : Bool -> Element msg -> Element msg
viewIf condition element =
    if condition then
        element

    else
        Element.none
