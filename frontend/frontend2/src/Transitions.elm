module Transitions exposing (transitions)

import Spa.Transition as Transition
import Utils.Spa as Spa


transitions : Spa.Transitions msg
transitions =
    { layout = Transition.fadeElmUi 300
    , page = Transition.fadeElmUi 100
    , pages = []
    }
