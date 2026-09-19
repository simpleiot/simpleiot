module UI.Layout exposing (layout)

import Element exposing (..)
import Element.Background as Background
import Element.Font as Font
import Gen.Route as Route
import UI.Form as Form
import UI.Style as Style


layout :
    { onSignOut : msg
    , email : Maybe String
    , error : Maybe String
    , badge : Maybe String
    }
    -> Element msg
    -> Element msg
layout options child =
    column [ spacing 32, padding 20, width (fill |> maximum 1280), height fill, centerX ]
        [ row
            [ width fill, spacing 20 ]
            [ logo
            , viewBadge options.badge
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


logo : Element msg
logo =
    Element.link []
        { label =
            image [ height (px 32) ]
                { src = "/siot-logo.svg"
                , description = "Simple IoT"
                }
        , url = Route.toHref Route.Home_
        }


viewError : Maybe String -> Element msg
viewError error =
    case error of
        Just err ->
            el Style.error (el [ centerX ] (text err))

        Nothing ->
            none


{-| viewBadge shows connection state next to the logo, such as while the
page is reconnecting to the server.
-}
viewBadge : Maybe String -> Element msg
viewBadge badge =
    case badge of
        Just label ->
            el
                [ Background.color Style.colors.ltblue
                , Font.size 12
                , paddingXY 6 2
                ]
            <|
                text label

        Nothing ->
            none
