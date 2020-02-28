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
    = SignedOut (Maybe Http.Error)
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
    ( SignedOut Nothing
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
        , url = "/v1/auth"
        , expect = Http.expectString (AuthResponse cred)
        }


update : Commands msg -> Msg -> Model -> ( Model, Cmd Msg, Cmd msg )
update commands msg model =
    case msg of
        SignIn cred ->
            ( SignedOut Nothing
            , login cred
            , commands.navigate routes.top
            )

        SignOut ->
            ( SignedOut Nothing
            , Cmd.none
            , Cmd.none
            )

        AuthResponse cred (Ok token) ->
            ( SignedIn
                { authToken = token
                , cred = cred
                }
            , Cmd.none
            , Cmd.none
            )


        AuthResponse cred (Err err)->
            ( SignedOut (Just err)
            , Cmd.none
            , commands.navigate routes.signIn
            )

subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none
