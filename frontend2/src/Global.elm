module Global exposing
    ( Flags
    , Model
    , Msg(..)
    , init
    , subscriptions
    , update
    )

import Generated.Routes as Routes exposing (Route, routes)
import Ports


type alias Flags =
    ()


type alias Model =
    { authToken: Maybe String
    , email: Maybe String}



type alias Cred =
    { email : String
    , password : String
    }


type Msg
    = SignIn Cred
    | SignOut


type alias Commands msg =
    { navigate : Route -> Cmd msg
    }


init : Commands msg -> Flags -> ( Model, Cmd Msg, Cmd msg )
init _ _ =
    ( { authToken = Nothing, email = Nothing}
    , Cmd.none
    , Ports.log "Hello!"
    )


update : Commands msg -> Msg -> Model -> ( Model, Cmd Msg, Cmd msg )
update commands msg model =
    case msg of
        SignIn cred ->
            ( {model | authToken = Just "hi there", email = Just cred.email}, Cmd.none, commands.navigate routes.top )
        SignOut ->
            ( {model | authToken = Nothing, email = Nothing}, Cmd.none, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none
