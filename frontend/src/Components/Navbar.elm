module Components.Navbar exposing (link, navbar, viewButtonLink)

import Element exposing (..)
import Spa.Generated.Route as Route exposing (Route)
import UI.Form as Form
import UI.Style as Style


navbar :
    { onSignOut : msg
    , authenticated : Bool
    , email : String
    }
    -> Element msg
navbar options =
    row [ width fill, spacing 20 ]
        [ link ( "SIOT", Route.Top )
        , el [ alignRight ] <|
            if options.authenticated then
                Form.button
                    { label = "sign out " ++ options.email
                    , color = Style.colors.blue
                    , onPress = options.onSignOut
                    }

            else
                Element.none
        ]


viewButtonLink : ( String, Route ) -> Element msg
viewButtonLink ( label, route ) =
    Element.link (Style.button Style.colors.blue)
        { label = text label
        , url = Route.toString route
        }


link : ( String, Route ) -> Element msg
link ( label, route ) =
    Element.link Style.link
        { label = text label
        , url = Route.toString route
        }
