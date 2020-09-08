module Shared exposing
    ( Flags
    , Model
    , Msg
    , init
    , subscriptions
    , update
    , view
    )

import Browser.Navigation exposing (Key)
import Data.Auth as Auth
import Element exposing (..)
import Element.Font as Font
import Spa.Document exposing (Document)
import Spa.Generated.Route as Route
import Url exposing (Url)



-- INIT


type alias Flags =
    ()


type alias Model =
    { url : Url
    , key : Key
    , authCred : Maybe Auth.Cred
    }


init : Flags -> Url -> Key -> ( Model, Cmd Msg )
init _ url key =
    ( Model url key Nothing
    , Cmd.none
    )



-- UPDATE


type Msg
    = SignOut


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        SignOut ->
            ( { model | authCred = Nothing }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none



-- VIEW


view :
    { page : Document msg, toMsg : Msg -> msg }
    -> Model
    -> Document msg
view { page, toMsg } model =
    { title = page.title
    , body =
        [ column [ padding 20, spacing 20, height fill ]
            [ row [ spacing 20 ]
                [ link [ Font.color (rgb 0 0.25 0.5), Font.underline ] { url = Route.toString Route.Top, label = text "Homepage" }
                , link [ Font.color (rgb 0 0.25 0.5), Font.underline ] { url = Route.toString Route.NotFound, label = text "Not found" }
                ]
            , column [ height fill ] page.body
            ]
        ]
    }
