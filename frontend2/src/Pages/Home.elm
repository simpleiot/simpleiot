module Pages.Home exposing (Model, Msg, page)

import Spa.Page
import Element exposing (..)
import Generated.Params as Params
import Utils.Spa exposing (Page)


type alias Model =
    ()


type alias Msg =
    Never


page : Page Params.Home Model Msg model msg appMsg
page =
    Spa.Page.static
        { title = always "Home"
        , view = always view
        }



-- VIEW


view : Element Msg
view =
    text "hi there"
