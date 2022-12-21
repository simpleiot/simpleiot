module UI.Layout exposing (layout)

import Element exposing (..)
import Gen.Route as Route exposing (Route)
import UI.Form as Form
import UI.Style as Style


layout :
    { onSignOut : msg
    , email : Maybe String
    , error : Maybe String
    }
    -> Element msg
    -> Element msg
layout options child =
    column [ spacing 32, padding 20, width (fill |> maximum 1280), height fill, centerX ]
        [ row
            [ width fill, spacing 20 ]
            [ link ( "SIOT", Route.Home_ )
            , el [ alignRight ] <|
                case options.email of
                    Just email_ ->
                        Form.button
                            { label = "sign out " ++ email_
                            , color = Style.colors.blue
                            , onPress = options.onSignOut
                            }

                    Nothing ->
                        Element.none
            ]
        , viewError options.error
        , child
        ]


link : ( String, Route ) -> Element msg
link ( label, route ) =
    Element.link Style.link
        { label = text label
        , url = Route.toHref route
        }


viewError : Maybe String -> Element msg
viewError error =
    case error of
        Just err ->
            el Style.error (el [ centerX ] (text err))

        Nothing ->
            none
