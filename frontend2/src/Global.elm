module Global exposing
    ( Flags
    , Model(..)
    , Msg(..)
    , init
    , subscriptions
    , update
    )

import Generated.Routes as Routes exposing (Route, routes)
import Ports
import Http


type alias Flags =
    ()


type Model
    = SignedOut
    | SignedIn
        { cred : Cred
        , authToken : String
        }


type alias Cred =
    { email : String
    , password : String
    }


type Msg
    = SignIn Cred
    | AuthResponse Cred (Result Http.Error String)
    | SignOut


type alias Commands msg =
    { navigate : Route -> Cmd msg
    }


init : Commands msg -> Flags -> ( Model, Cmd Msg, Cmd msg )
init _ _ =
    ( SignedOut
    , Cmd.none
    , Ports.log "Hello!"
    )


login : Cred -> Cmd Msg
login cred =
    Http.post
        { body = Http.multipartBody
            [ Http.stringPart "email" cred.email
            , Http.stringPart "password" cred.password
            ]
        , url = "http://localhost:8080/v1/auth"
        , expect = Http.expectString (\resp -> AuthResponse cred resp)
        }


update : Commands msg -> Msg -> Model -> ( Model, Cmd Msg, Cmd msg )
update commands msg model =
    case msg of
        SignIn cred ->
            ( SignedOut
            , login cred
            , commands.navigate routes.top
            )

        SignOut ->
            ( SignedOut
            , Cmd.none
            , Cmd.none
            )

        AuthResponse cred resp ->
            ( case resp of
                Ok token ->
                    SignedIn
                        { authToken = token
                        , cred = cred
                        }

                Err err ->
                    -- TODO: display an error message
                    SignedOut

            , Cmd.none
            , Cmd.none
            )

subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none
