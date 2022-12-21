module Pages.NotFound exposing (page)

import Gen.Params.NotFound exposing (Params)
import Page exposing (Page)
import Request
import Shared
import View exposing (View)


page : Shared.Model -> Request.With Params -> Page
page _ _ =
    Page.static
        { view = view
        }


view : View msg
view =
    View.placeholder "NotFound"
