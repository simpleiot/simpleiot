module Pages.Home_ exposing (Model, Msg, page)

import Auth
import Effect exposing (Effect)
import Element exposing (..)
import Gen.Params.Home_ exposing (Params)
import Page
import Request
import Shared
import Storage
import UI.Layout
import View exposing (View)


page : Shared.Model -> Request.With Params -> Page.With Model Msg
page shared req =
    Page.protected.advanced <|
        \user ->
            { init = init
            , update = update shared
            , view = view user shared
            , subscriptions = subscriptions
            }



-- INIT


type alias Model =
    {}


init : ( Model, Effect Msg )
init =
    ( {}, Effect.none )



-- UPDATE


type Msg
    = SignOut


update : Shared.Model -> Msg -> Model -> ( Model, Effect Msg )
update shared msg model =
    case msg of
        SignOut ->
            ( model, Effect.fromCmd <| Storage.signOut shared.storage )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- VIEW


view : Auth.User -> Shared.Model -> Model -> View Msg
view user shared model =
    { title = "SIOT"
    , attributes = []
    , element =
        UI.Layout.layout
            { onSignOut = SignOut
            , email =
                case shared.storage.user of
                    Just user_ ->
                        Just user_.email

                    Nothing ->
                        Nothing
            , error = shared.error
            }
            (text "Home_")
    }
