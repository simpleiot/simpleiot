module Global exposing
    ( Flags
    , Model(..)
    , Msg(..)
    , init
    , subscriptions
    , update
    )

import Device as D
import Generated.Routes exposing (Route, routes)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required, resolve)
import Json.Encode as Encode
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
    , respError : Maybe String
    , posting : Bool
    }


emptyData : Data
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
    | UsersResponse (Result Http.Error (List U.User))
    | SignIn Cred
    | AuthResponse Cred (Result Http.Error Auth)
    | DataResponse (Result Http.Error Data)
    | RequestOrgs
    | RequestDevices
    | RequestUsers
    | SignOut
    | Tick Time.Posix
    | UpdateDeviceConfig String D.Config
    | UpdateUser U.User
    | ConfigPosted String (Result Http.Error Response)
    | UserPosted String (Result Http.Error Response)


type alias Commands msg =
    { navigate : Route -> Cmd msg
    }


init : Commands msg -> Flags -> ( Model, Cmd Msg, Cmd msg )
init _ _ =
    ( SignedOut Nothing
    , Cmd.none
    , Cmd.none
    )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.batch
        [ Time.every 10000 Tick
        ]


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


type alias Auth =
    { token : String
    , privilege : Privilege
    }


type Privilege
    = User
    | Admin
    | Root


decodeAuth : Decode.Decoder Auth
decodeAuth =
    Decode.succeed validate
        |> required "token" Decode.string
        |> required "privilege" Decode.string
        |> resolve


validate : String -> String -> Decode.Decoder Auth
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
                        , respError = Nothing
                        , posting = False
                        }
                    , Cmd.none
                    , commands.navigate routes.top
                    )

                AuthResponse _ (Err error) ->
                    ( SignedOut (Just error), Cmd.none, Cmd.none )

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

                AuthResponse _ (Err err) ->
                    ( SignedOut (Just err)
                    , Cmd.none
                    , commands.navigate routes.signIn
                    )

                DataResponse (Ok newData) ->
                    ( SignedIn { sess | data = newData }
                    , Cmd.none
                    , Cmd.none
                    )

                DataResponse (Err _) ->
                    ( SignedIn { sess | respError = Just "Error getting data" }
                    , Cmd.none
                    , Cmd.none
                    )

                DevicesResponse (Ok devices) ->
                    ( SignedIn
                        { sess
                            | data = { data | devices = devices }
                        }
                    , Cmd.none
                    , Cmd.none
                    )

                DevicesResponse (Err _) ->
                    ( SignedIn
                        { sess
                            | respError = Just "Error getting devices"
                        }
                    , Cmd.none
                    , Cmd.none
                    )

                UsersResponse (Ok users) ->
                    ( SignedIn { sess | data = { data | users = users } }
                    , Cmd.none
                    , Cmd.none
                    )

                UsersResponse (Err _) ->
                    ( SignedIn { sess | respError = Just "Error getting users" }
                    , Cmd.none
                    , Cmd.none
                    )

                RequestDevices ->
                    ( model
                    , if sess.posting then
                        Cmd.none

                      else
                        getDevices sess.authToken
                    , Cmd.none
                    )

                RequestUsers ->
                    ( model
                    , getUsers sess.authToken
                    , Cmd.none
                    )

                OrgsResponse (Ok orgs) ->
                    ( SignedIn { sess | data = { data | orgs = orgs } }
                    , Cmd.none
                    , Cmd.none
                    )

                OrgsResponse (Err _) ->
                    ( SignedIn { sess | respError = Just "Error getting orgs" }
                    , Cmd.none
                    , Cmd.none
                    )

                RequestOrgs ->
                    ( model
                    , getOrgs sess.authToken
                    , Cmd.none
                    )

                Tick _ ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )

                UpdateDeviceConfig id config ->
                    let
                        updateConfig device =
                            if device.id == id then
                                { device | config = config }

                            else
                                device

                        devices =
                            List.map updateConfig sess.data.devices

                        oldData =
                            sess.data

                        newData =
                            { oldData | devices = devices }
                    in
                    ( SignedIn { sess | data = newData, posting = True }
                    , postConfig sess.authToken id config
                    , Cmd.none
                    )

                UpdateUser user ->
                    let
                        -- update local model to make UI optimistic
                        updateUser old =
                            if old.id == user.id then
                                user

                            else
                                old

                        users =
                            if user.id == "" then
                                [ user ] ++ sess.data.users

                            else
                                List.map updateUser sess.data.users
                    in
                    ( SignedIn { sess | data = { data | users = users } }
                    , postUser sess.authToken user
                    , Cmd.none
                    )

                ConfigPosted _ (Ok _) ->
                    ( SignedIn { sess | posting = False }
                    , Cmd.none
                    , Cmd.none
                    )

                ConfigPosted _ (Err _) ->
                    ( SignedIn
                        { sess
                            | respError = Just "Error saving device config"
                            , posting = False
                        }
                    , Cmd.none
                    , Cmd.none
                    )

                UserPosted _ (Err _) ->
                    ( SignedIn { sess | respError = Just "Error saving user" }
                    , Cmd.none
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


type alias Response =
    { success : Bool
    , error : String
    , id : String
    }


deviceConfigEncoder : D.Config -> Encode.Value
deviceConfigEncoder deviceConfig =
    Encode.object
        [ ( "description", Encode.string deviceConfig.description ) ]


responseDecoder : Decode.Decoder Response
responseDecoder =
    Decode.succeed Response
        |> required "success" Decode.bool
        |> optional "error" Decode.string ""
        |> optional "id" Decode.string ""


postConfig : String -> String -> D.Config -> Cmd Msg
postConfig token id config =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "devices", id, "config" ] []
        , expect = Http.expectJson (ConfigPosted id) responseDecoder
        , body = config |> deviceConfigEncoder |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


urlDevices : String
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


getUsers : String -> Cmd Msg
getUsers token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "users" ] []
        , expect = Http.expectJson UsersResponse U.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


postUser : String -> U.User -> Cmd Msg
postUser token user =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "users", user.id ] []
        , expect = Http.expectJson (UserPosted user.id) responseDecoder
        , body = user |> U.encode |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }
