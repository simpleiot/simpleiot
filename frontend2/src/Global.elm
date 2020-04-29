module Global exposing
    ( Flags
    , Model(..)
    , Msg(..)
    , init
    , subscriptions
    , update
    )

import Device as D
import Generated.Routes as Routes exposing (Route, routes)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (hardcoded, optional, required, resolve)
import Org as O
import Time
import Url.Builder as Url
import User as U


type alias Flags =
    ()


type Model
    = SignedOut (Maybe Http.Error)
    | SignedIn Session


type alias Session =
    { cred : Cred
    , authToken : String
    , privilege : Privilege
    , data : Data
    , error : Maybe Http.Error
    }


emptyData =
    { orgs = []
    , users = []
    , devices = []
    }


type alias Data =
    { orgs : List O.Org
    , devices : List D.Device
    , users : List U.User
    }


type alias Cred =
    { email : String
    , password : String
    }


type Msg
    = DevicesResponse (Result Http.Error (List D.Device))
    | OrgsResponse (Result Http.Error (List O.Org))
    | SignIn Cred
    | AuthResponse Cred (Result Http.Error Auth)
    | DataResponse (Result Http.Error Data)
    | RequestOrgs
    | RequestDevices
    | SignOut


type alias Commands msg =
    { navigate : Route -> Cmd msg
    }


init : Commands msg -> Flags -> ( Model, Cmd Msg, Cmd msg )
init _ _ =
    ( SignedOut Nothing
    , Cmd.none
    , Cmd.none
    )


login : Cred -> Cmd Msg
login cred =
    Http.post
        { body =
            Http.multipartBody
                [ Http.stringPart "email" cred.email
                , Http.stringPart "password" cred.password
                ]
        , url = Url.absolute [ "v1", "auth" ] []
        , expect = Http.expectJson (AuthResponse cred) decodeAuth
        }


getData token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "data" ] []
        , expect = Http.expectJson DataResponse decodeData
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


decodeData =
    Decode.succeed Data
        |> required "orgs" O.decodeList
        |> required "devices" D.decodeList
        |> required "users" U.decodeList


type alias Auth =
    { token : String
    , privilege : Privilege
    }


type Privilege
    = User
    | Admin
    | Root


decodeAuth =
    Decode.succeed validate
        |> required "token" Decode.string
        |> required "privilege" Decode.string
        |> resolve


validate token privilege =
    case privilege of
        "user" ->
            Decode.succeed <| Auth token User

        "admin" ->
            Decode.succeed <| Auth token Admin

        "root" ->
            Decode.succeed <| Auth token Root

        _ ->
            Decode.fail "sign in failed"


update : Commands msg -> Msg -> Model -> ( Model, Cmd Msg, Cmd msg )
update commands msg model =
    case model of
        SignedOut _ ->
            case msg of
                SignIn cred ->
                    ( SignedOut Nothing
                    , login cred
                    , Cmd.none
                    )

                AuthResponse cred (Ok { token, privilege }) ->
                    ( SignedIn
                        { authToken = token
                        , cred = cred
                        , privilege = privilege
                        , data = emptyData
                        , error = Nothing
                        }
                    , getData token
                    , commands.navigate routes.top
                    )

                AuthResponse cred (Err error) ->
                    let
                        _ = Debug.log "Auth error" error
                    in
                        (SignedOut (Just error), Cmd.none, Cmd.none)


                _ ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )

        SignedIn sess ->
            let
                data =
                    sess.data
            in
            case msg of
                SignOut ->
                    ( SignedOut Nothing
                    , Cmd.none
                    , commands.navigate routes.top
                    )

                AuthResponse cred (Err err) ->
                    ( SignedOut (Just err)
                    , Cmd.none
                    , commands.navigate routes.signIn
                    )

                DataResponse (Ok newData) ->
                    ( SignedIn { sess | data = newData }
                    , Cmd.none
                    , Cmd.none
                    )

                DevicesResponse (Ok devices) ->
                    ( SignedIn { sess | data = { data | devices = devices } }
                    , Cmd.none
                    , Cmd.none
                    )

                OrgsResponse (Ok orgs) ->
                    ( SignedIn { sess | data = { data | orgs = orgs } }
                    , Cmd.none
                    , Cmd.none
                    )

                RequestOrgs ->
                    ( model
                    , getOrgs sess.authToken
                    , Cmd.none
                    )

                RequestDevices ->
                    ( model
                    , getDevices sess.authToken
                    , Cmd.none
                    )

                _ ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )


getDevices : String -> Cmd Msg
getDevices token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = urlDevices
        , expect = Http.expectJson DevicesResponse D.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


urlDevices =
    Url.absolute [ "v1", "devices" ] []


getOrgs : String -> Cmd Msg
getOrgs token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "orgs" ] []
        , expect = Http.expectJson OrgsResponse O.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none
