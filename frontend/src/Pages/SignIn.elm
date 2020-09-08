module Pages.SignIn exposing (Model, Msg, Params, page)

import Api.Auth as Auth
import Api.Data exposing (Data)
import Browser.Navigation exposing (Key)
import Shared
import Spa.Document exposing (Document)
import Spa.Page as Page exposing (Page)
import Spa.Url as Url exposing (Url)


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
    { authCred : Data Auth.Cred
    , key : Key
    , email : String
    , password : String
    }


init : Shared.Model -> Url Params -> ( Model, Cmd Msg )
init shared { key } =
    ( Model
        (case shared.authCred of
            Just cred ->
                Api.Data.Success cred

            Nothing ->
                Api.Data.NotAsked
        )
        key
        ""
        ""
    , Cmd.none
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
save _ shared =
    shared


load : Shared.Model -> Model -> ( Model, Cmd Msg )
load _ model =
    ( model, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- VIEW


view : Model -> Document Msg
view model =
    { title = "SignIn"
    , body = []
    }
