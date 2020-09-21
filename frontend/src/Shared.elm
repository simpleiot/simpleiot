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
import Components.Navbar exposing (navbar)
import Element exposing (..)
import Spa.Document exposing (Document)
import Url exposing (Url)



-- INIT


type alias Flags =
    ()


type alias Auth =
    { email : String
    , token : String
    , isRoot : Bool
    }


type alias Model =
    { url : Url
    , key : Key
    , auth : Maybe Auth
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
            ( { model | auth = Nothing }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none



-- VIEW


view :
    { page : Document msg, toMsg : Msg -> msg }
    -> Model
    -> Document msg
view { page, toMsg } model =
    let
        ( authenticated, isRoot, email ) =
            case model.auth of
                Just auth ->
                    ( True, auth.isRoot, auth.email )

                Nothing ->
                    ( False, False, "" )
    in
    { title = page.title
    , body =
        [ column [ padding 20, spacing 20, height fill ]
            [ navbar
                { onSignOut = toMsg SignOut
                , authenticated = authenticated
                , isRoot = isRoot
                , email = email
                }
            , column [ height fill ] page.body
            ]
        ]
    }
