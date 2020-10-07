module Pages.Msg exposing (Model, Msg, Params, page)

import Api.Auth exposing (Auth)
import Shared
import Spa.Document exposing (Document)
import Spa.Generated.Route as Route
import Spa.Page as Page exposing (Page)
import Spa.Url exposing (Url)
import Utils.Route


page : Page Params Model Msg
page =
    Page.application
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        , save = save
        , load = load
        }



-- INIT


type alias Params =
    ()


type alias Model =
    { auth : Auth
    }


defaultModel : Model
defaultModel =
    Model
        { email = "", token = "", isRoot = False }


init : Shared.Model -> Url Params -> ( Model, Cmd Msg )
init shared { params } =
    case shared.auth of
        Just auth ->
            let
                model =
                    { defaultModel | auth = auth }
            in
            ( model
            , Cmd.none
            )

        Nothing ->
            ( defaultModel
            , Utils.Route.navigate shared.key Route.SignIn
            )



-- UPDATE


type Msg
    = ReplaceMe


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        ReplaceMe ->
            ( model, Cmd.none )


save : Model -> Shared.Model -> Shared.Model
save model shared =
    shared


load : Shared.Model -> Model -> ( Model, Cmd Msg )
load shared model =
    ( model, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- VIEW


view : Model -> Document Msg
view model =
    { title = "Msg"
    , body = []
    }
