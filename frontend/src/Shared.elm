module Shared exposing
    ( Flags
    , Model
    , Msg
    , init
    , subscriptions
    , update
    , view
    )

import Api.Auth exposing (Auth)
import Browser.Navigation exposing (Key)
import Components.Navbar exposing (navbar)
import Element exposing (..)
import Spa.Document exposing (Document)
import Spa.Generated.Route as Route
import Time
import UI.Style as Style
import Url exposing (Url)
import Utils.Route



-- INIT


type alias Flags =
    ()


type alias Model =
    { url : Url
    , key : Key
    , auth : Maybe Auth
    , error : Maybe String
    , now : Time.Posix
    , lastError : Time.Posix
    }


init : Flags -> Url -> Key -> ( Model, Cmd Msg )
init _ url key =
    ( Model url key Nothing Nothing (Time.millisToPosix 0) (Time.millisToPosix 0)
    , Cmd.none
    )



-- UPDATE


type Msg
    = SignOut
    | Tick Time.Posix


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        SignOut ->
            ( { model | auth = Nothing }
            , Utils.Route.navigate model.key Route.SignIn
            )

        Tick now ->
            let
                error =
                    if Time.posixToMillis now - Time.posixToMillis model.lastError > 5 * 1000 then
                        Nothing

                    else
                        model.error
            in
            ( { model | now = now, error = error }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.batch
        [ Time.every 1000 Tick
        ]



-- VIEW


view :
    { page : Document msg, toMsg : Msg -> msg }
    -> Model
    -> Document msg
view { page, toMsg } model =
    let
        ( authenticated, email ) =
            case model.auth of
                Just auth ->
                    ( True, auth.email )

                Nothing ->
                    ( False, "" )
    in
    { title = page.title
    , body =
        [ column [ spacing 32, padding 20, width (fill |> maximum 1280), height fill, centerX ]
            [ navbar
                { onSignOut = toMsg SignOut
                , authenticated = authenticated
                , email = email
                }
            , viewError model.error
            , column [ height fill ] page.body
            ]
        ]
    }


viewError : Maybe String -> Element msg
viewError error =
    case error of
        Just err ->
            el Style.error (el [ centerX ] (text err))

        Nothing ->
            none
