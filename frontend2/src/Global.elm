module Global exposing
    ( Flags
    , Model
    , Msg(..)
    , init
    , subscriptions
    , update
    )

import Generated.Routes as Routes exposing (Route)
import Ports


type alias Flags =
    ()


type alias Model =
    {}


type alias Cred =
    { email : String
    , password : String
    }


type Msg
    = Login Cred


type alias Commands msg =
    { navigate : Route -> Cmd msg
    }


init : Commands msg -> Flags -> ( Model, Cmd Msg, Cmd msg )
init _ _ =
    ( {}
    , Cmd.none
    , Ports.log "Hello!"
    )


update : Commands msg -> Msg -> Model -> ( Model, Cmd Msg, Cmd msg )
update _ msg model =
    case msg of
        Login cred ->
            ( model, Cmd.none, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none
